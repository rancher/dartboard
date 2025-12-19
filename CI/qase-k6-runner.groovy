#!groovy
// Declarative Pipeline Syntax
@Library('qa-jenkins-library') _

def agentLabel = 'jenkins-qa-jenkins-agent'
if (params.JENKINS_AGENT_LABEL) {
  agentLabel = params.JENKINS_AGENT_LABEL
}

def kubeconfigContainerPath
def baseURL
def sanitizeCharacterRegex = "[^a-zA-Z0-9'_-]"

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
              // Sanitize inputs to prevent shell injection
              def safeRegion = (params.S3_BUCKET_REGION ?: "").replaceAll(sanitizeCharacterRegex, "")
              def safeBucket = (params.S3_BUCKET_NAME ?: "").replaceAll(sanitizeCharacterRegex, "")
              def safeDeploymentId = (params.DEPLOYMENT_ID ?: "").replaceAll(sanitizeCharacterRegex, "")

              withEnv(["SAFE_REGION=${safeRegion}", "SAFE_BUCKET=${safeBucket}", "SAFE_DEPLOYMENT_ID=${safeDeploymentId}"]) {
                sh """
                  mkdir -p ${env.ARTIFACTS_DIR}
                  docker run --rm \\
                      -v "${pwd()}/${env.ARTIFACTS_DIR}:/artifacts" \\
                      -e AWS_ACCESS_KEY_ID \\
                      -e AWS_SECRET_ACCESS_KEY \\
                      -e AWS_S3_REGION="\${SAFE_REGION}" \\
                      amazon/aws-cli s3 cp "s3://\${SAFE_BUCKET}/\${SAFE_DEPLOYMENT_ID}/" /artifacts/ --recursive

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
            // Sanitize Run ID to be numeric only to prevent command injection
            def safeRunID = (params.QASE_TESTOPS_RUN_ID ?: "").replaceAll("[^0-9]", "")

            withCredentials([string(credentialsId: "QASE_AUTOMATION_TOKEN", variable: "QASE_TESTOPS_API_TOKEN")]) {
              sh """
                docker run --rm --name dartboard-qase-gatherer \\
                  -e QASE_TESTOPS_API_TOKEN \\
                  -e QASE_TESTOPS_PROJECT="${params.QASE_TESTOPS_PROJECT}" \\
                  ${env.IMAGE_NAME}:latest qase-k6-cli gather -runID "${safeRunID}" > test_cases.json
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

              // Sanitize project name to prevent shell injection in filenames
              def safeProject = (params.QASE_TESTOPS_PROJECT ?: "").replaceAll(sanitizeCharacterRegex, "")

              // 1. Prepare Environment for this specific test case
              // Use index to ensure uniqueness for file names when multiple parameter combinations exist for the same case ID
              def envFile = "k6-${caseId}-${index}.env"
              def k6Test = "${scriptPath}"
              def summaryLog = "k6-summary-params-${safeProject}-${caseId}-${index}.log"
              def summaryJson = "k6-summary-params-${safeProject}-${caseId}-${index}.json"
              def htmlReport = "k6-report-${safeProject}-${caseId}-${index}.html"
              def webDashboardReport = "k6-web-dashboard-${safeProject}-${caseId}-${index}.html"

              // Construct environment variables content
              // We set QASE_TEST_CASE_ID for the reporter
              def envContent = """
K6_NO_USAGE_REPORT=true
K6_TEST=${k6Test}
BASE_URL=${baseURL ?: ''}
KUBECONFIG=${kubeconfigContainerPath ?: ''}
QASE_TESTOPS_PROJECT="${params.QASE_TESTOPS_PROJECT}"
QASE_TESTOPS_RUN_ID="${params.QASE_TESTOPS_RUN_ID}"
QASE_TEST_CASE_ID=${caseId}
K6_SUMMARY_JSON_FILE=${summaryJson}
K6_HTML_REPORT_FILE=${htmlReport}
K6_WEB_DASHBOARD=true
K6_WEB_DASHBOARD_EXPORT=${webDashboardReport}
"""
              // Handle parameters required by the test case
              parameters.each { paramName, paramValue ->
                 // Check if the Jenkins job has this parameter defined, otherwise use the value from Qase
                 def finalValue = params[paramName] ?: paramValue
                 // Sanitize value to prevent newlines breaking the env file format
                 finalValue = finalValue.toString().replaceAll("[\r\n]", "")
                 envContent += "${paramName}=${finalValue}\n"
              }

              writeFile file: envFile, text: envContent

              // 2. Run k6
              try {
                  sh """
                    docker run --rm --name dartboard-k6-runner-${index} \\
                        -v "${pwd()}:/app" \\
                        --env-file "${envFile}" \\
                        --workdir /app \\
                        --user=\$(id -u) \\
                        --entrypoint='' \\
                        ${env.IMAGE_NAME}:latest sh -c '''
                            echo "Running k6 script: \$K6_TEST"
                            # Ensure the directory for the summary file exists or k6 might complain if path is deep
                            # Run k6, piping output to log and generating summary JSON
                            k6 run --no-color "\$K6_TEST" > "${summaryLog}" 2>&1
                        '''
                  """
              } catch (Exception e) {
                  echo "k6 run failed for case ${caseId}, but continuing to report failure/partial results."
              }

              // 3. Report to Qase
              withCredentials([string(credentialsId: "QASE_AUTOMATION_TOKEN", variable: "QASE_TESTOPS_API_TOKEN")]) {
                  sh """
                    docker run --rm --name dartboard-qase-reporter-${index} \\
                        -v "${pwd()}:/app" \\
                        --env-file "${envFile}" \\
                        --workdir /app \\
                        --user=\$(id -u) \\
                        --entrypoint='' \\
                        -e QASE_TESTOPS_API_TOKEN \\
                        ${env.IMAGE_NAME}:latest sh -c '''
                            echo "Reporting results for Case ${caseId}..."
                            if [ -f "${summaryJson}" ]; then
                                qase-k6-cli report
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
        echo "Archiving k6 test results..."
        archiveArtifacts artifacts: """
          dartboard/*.json,
          dartboard/*.log,
          dartboard/*.html,
          dartboard/*.xml,
        """.trim(), fingerprint: true

        // The k6 container is run with --rm, so it should clean itself up.
        // But if the job is aborted, the container might be left running.
        echo "Cleaning up Docker resources..."
        try {
          echo "Attempting to remove containers matching: dartboard-k6-runner"
          sh "docker ps -a -q --filter name=dartboard-k6-runner | xargs -r docker rm -f"
        } catch (e) {
          echo "Could not remove containers matching 'dartboard-k6-runner'. Details: ${e.message}"
        }
        try {
          echo "Attempting to remove container: dartboard-qase-gatherer"
          sh "docker rm -f dartboard-qase-gatherer"
        } catch (e) {
          echo "Could not remove container 'dartboard-qase-gatherer'. It may have already been removed. Details: ${e.message}"
        }
        try {
          echo "Attempting to remove containers matching: dartboard-qase-reporter"
          sh "docker ps -a -q --filter name=dartboard-qase-reporter | xargs -r docker rm -f"
        } catch (e) {
          echo "Could not remove containers matching 'dartboard-qase-reporter'. Details: ${e.message}"
        }
        try {
          echo "Attempting to remove image: ${env.IMAGE_NAME}:latest"
          sh "docker rmi -f ${env.IMAGE_NAME}:latest"
          echo "Attempting to remove image: amazon/aws-cli"
          sh "docker rmi amazon/aws-cli"
        } catch (e) {
          echo "Could not remove a Docker image. It may have already been removed or was never present. Details: ${e.message}"
        }
      }
    }
    cleanup {
      // Clean up large files from the workspace to save disk space on the agent.
      // These are not part of the archived artifacts but remain in the workspace.
      echo "Cleaning up workspace..."
      dir('dartboard') {
        // Use find and xargs for more robust and efficient cleanup of non-artifact files and directories.
        // This removes all files and directories from the checkout except for the archived k6 results.
        sh """
          set -x
          echo "Removing all non-artifact files and directories..."
          find . -mindepth 1 -maxdepth 1 \\
            -not -name '*.html' -not -name '*.json' -not -name '*.log' -not -name '*.xml' \\
            -exec rm -rf {} +
        """
      }
    }
  }
}
