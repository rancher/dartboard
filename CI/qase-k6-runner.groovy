#!groovy
// Declarative Pipeline Syntax
@Library('qa-jenkins-library') _

def agentLabel = 'jenkins-qa-jenkins-agent'
if (params.JENKINS_AGENT_LABEL) {
  agentLabel = params.JENKINS_AGENT_LABEL
}

def kubeconfigContainerPath
def baseURL

pipeline {
  agent { label agentLabel }

  environment {
    IMAGE_NAME          = 'dartboard'
    ARTIFACTS_DIR       = 'deployment-artifacts'
    ACCESS_LOG          = 'access-details.log'
    KUBECONFIG_FILE     = 'upstream.yaml'
    // QASE_TESTOPS_PROJECT and QASE_TESTOPS_RUN_ID are expected as build parameters
  }

  stages {
    stage('Checkout') {
      steps {
        script {
          project.checkout(repository: params.REPO, branch: params.BRANCH, target: 'dartboard')
        }
      }
    }

    stage('Set Build Description') {
      steps {
        script {
          def testRun = params.QASE_TESTOPS_PROJECT && params.QASE_TESTOPS_RUN_ID ? "${params.QASE_TESTOPS_PROJECT}-${params.QASE_TESTOPS_RUN_ID}": ''
          currentBuild.description = "${testRun}"
        }
      }
    }

    stage('Prepare Environment from S3') {
      when { expression { return params.DEPLOYMENT_ID } }
      steps {
        dir('dartboard') {
          script {
            property.useWithCredentials(['AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY']) {
              sh """
                mkdir -p ${env.ARTIFACTS_DIR}
                docker run --rm \\
                    -v "${pwd()}/${env.ARTIFACTS_DIR}:/artifacts" \\
                    -e AWS_ACCESS_KEY_ID \\
                    -e AWS_SECRET_ACCESS_KEY \\
                    -e AWS_S3_REGION="${params.S3_BUCKET_REGION}" \\
                    amazon/aws-cli s3 cp "s3://${params.S3_BUCKET_NAME}/${params.DEPLOYMENT_ID}/" /artifacts/ --recursive

                # Unzip the config archive
                config_zip=\$(find ${env.ARTIFACTS_DIR} -name '*_config.zip' | head -n 1)
                if [ -n "\$config_zip" ]; then
                  unzip -o "\$config_zip" -d "${env.ARTIFACTS_DIR}"
                else
                  echo "Warning: No config zip file found in S3 artifacts."
                fi

                echo "Downloaded artifacts:"
                ls -l ${env.ARTIFACTS_DIR}
              """
            }
            // Extract FQDN and set environment variables for the next stage
            def accessLogPath = "./${env.ARTIFACTS_DIR}/${env.ACCESS_LOG}"
            if (fileExists(accessLogPath)) {
              def accessLogContent = readFile(accessLogPath)
              // See https://docs.groovy-lang.org/next/html/groovy-jdk/java/util/regex/Matcher.html
              def matcher = accessLogContent =~ /(?m)^\s*Rancher UI:\s*(https?:\/\/[^ :]+)/
              if (matcher.find()) {
                def match = matcher.group(1).trim()
                baseURL = "${match}"
                echo "Found Rancher URL: ${baseURL}"
              } else {
                echo "Warning: Could not find 'Rancher UI' in ${env.ACCESS_LOG}"
              }
            }

            // Find the upstream.yaml file within the downloaded artifacts and move it to the current directory.
            // This is more robust than assuming its exact location after unzipping.
            sh """
              kubeconfig_file=\$(find ./${env.ARTIFACTS_DIR} -name '${env.KUBECONFIG_FILE}' -print -quit)
              if [ -n "\$kubeconfig_file" ]; then
                mv "\$kubeconfig_file" "./${env.KUBECONFIG_FILE}"
              fi
            """
            def kubeconfigPath = "./${env.KUBECONFIG_FILE}"
            if (fileExists(kubeconfigPath)) {
              // Absolute path relative to the container's filespace
              kubeconfigContainerPath = "/app/${env.KUBECONFIG_FILE}"
              echo "Found kubeconfig at: ${kubeconfigPath}"
            }
          }
        }
      }
    }

    stage('Build Dartboard Image') {
      steps {
        dir('dartboard') {
          sh "docker build -t ${env.IMAGE_NAME}:latest ."
        }
      }
    }

    stage('Gather Test Cases') {
      steps {
        dir('dartboard') {
          script {
            withCredentials([string(credentialsId: "QASE_AUTOMATION_TOKEN", variable: "QASE_TESTOPS_API_TOKEN")]) {
              sh """
                docker run --rm \\
                  -e QASE_TESTOPS_API_TOKEN \\
                  -e QASE_TESTOPS_PROJECT="${params.QASE_TESTOPS_PROJECT}" \\
                  ${env.IMAGE_NAME}:latest /app/qase-k6-cli gather -runID ${params.QASE_TESTOPS_RUN_ID} > test_cases.json
              """
            }
            sh "cat test_cases.json"
          }
        }
      }
    }

    stage('Run Tests') {
      steps {
        dir('dartboard') {
          script {
            def testCases = readJSON file: 'test_cases.json'

            if (testCases.size() == 0) {
                echo "No test cases found with 'AutomationTestName' custom field in Run ID ${params.QASE_TESTOPS_RUN_ID}."
                return
            }

            // Iterate over each gathered test case
            testCases.eachWithIndex { testCase, index ->
              def caseId = testCase.id
              def caseTitle = testCase.title
              def scriptPath = testCase.automation_test_name
              def parameters = testCase.parameters // Map of parameter name -> value

              echo "-------------------------------------------------------"
              echo "Processing Case ID: ${caseId}"
              echo "Title: ${caseTitle}"
              echo "Script: ${scriptPath}"
              echo "Parameters: ${parameters}"
              echo "-------------------------------------------------------"

              // 1. Prepare Environment for this specific test case
              // Use index to ensure uniqueness for file names when multiple parameter combinations exist for the same case ID
              def envFile = "k6-${caseId}-${index}.env"
              def summaryLog = "k6-summary-${caseId}-${index}.log"
              def summaryJson = "k6-summary-${caseId}-${index}.json"
              def htmlJson = "k6-report-${caseId}-${index}.html"

              // Construct environment variables content
              // We set QASE_TEST_CASE_ID for the reporter
              def envContent = """
K6_NO_USAGE_REPORT=true
BASE_URL=${baseURL ?: ''}
KUBECONFIG=${kubeconfigContainerPath ?: ''}
QASE_TEST_CASE_ID=${caseId}
K6_SUMMARY_JSON_FILE=${summaryJson}
K6_HTML_REPORT_FILE=${htmlJson}
"""
              // Handle parameters required by the test case
              parameters.each { paramName, paramValue ->
                 // Check if the Jenkins job has this parameter defined, otherwise use the value from Qase
                 def finalValue = params[paramName] ?: paramValue
                 envContent += "${paramName}=${finalValue}\n"
              }

              writeFile file: envFile, text: envContent

              // 2. Run k6
              try {
                  sh """
                    docker run --rm \\
                        -v "${pwd()}:/app" \\
                        --workdir /app \\
                        --user=\$(id -u) \\
                        --entrypoint='' \\
                        ${env.IMAGE_NAME}:latest sh -c '''
                            set -o allexport
                            source "${envFile}"
                            set +o allexport

                            echo "Running k6 script: ${scriptPath}"
                            # Ensure the directory for the summary file exists or k6 might complain if path is deep
                            # Run k6, piping output to log and generating summary JSON
                            k6 run --no-color "${scriptPath}" > "${summaryLog}" 2>&1
                        '''
                  """
              } catch (Exception e) {
                  echo "k6 run failed for case ${caseId}, but continuing to report failure/partial results."
              }

              // 3. Report to Qase
              withCredentials([string(credentialsId: "QASE_AUTOMATION_TOKEN", variable: "QASE_TESTOPS_API_TOKEN")]) {
                  sh """
                    docker run --rm \\
                        -v "${pwd()}:/app" \\
                        --workdir /app \\
                        --user=\$(id -u) \\
                        --entrypoint='' \\
                        -e QASE_TESTOPS_API_TOKEN \\
                        -e QASE_TESTOPS_PROJECT="${params.QASE_TESTOPS_PROJECT}" \\
                        -e QASE_TESTOPS_RUN_ID="${params.QASE_TESTOPS_RUN_ID}" \\
                        ${env.IMAGE_NAME}:latest sh -c '''
                            set -o allexport
                            source "${envFile}"
                            set +o allexport

                            echo "Reporting results for Case ${caseId}..."
                            if [ -f "${summaryJson}" ]; then
                                /app/qase-k6-cli report
                            else
                                echo "Summary JSON not found, skipping report for ${caseId}"
                            fi
                        '''
                  """
              }
            }
          }
        }
      }
    }
  }

  post {
    always {
      script {
        archiveArtifacts artifacts: 'dartboard/*.log, dartboard/*.json', fingerprint: true, allowEmptyArchive: true
        sh "docker rmi -f ${env.IMAGE_NAME}:latest || true"
      }
    }
  }
}
