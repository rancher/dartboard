#!groovy
// Declarative Pipeline Syntax
@Library('qa-jenkins-library') _

def agentLabel = 'jenkins-qa-jenkins-agent'
if (params.JENKINS_AGENT_LABEL) {
  agentLabel = params.JENKINS_AGENT_LABEL
}

def testFileBasename
def kubeconfigContainerPath
def baseURL

pipeline {
  agent { label agentLabel }

  environment {
    IMAGE_NAME          = 'dartboard'
    K6_ENV_FILE         = 'k6.env'
    K6_SUMMARY_LOG      = 'k6-summary.log'
    S3_ARTIFACT_PREFIX  = "${JOB_NAME.split('/').last()}-${BUILD_NUMBER}"
    ARTIFACTS_DIR       = 'deployment-artifacts'
    ACCESS_LOG          = 'access-details.log'
    KUBECONFIG_FILE     = 'upstream.yaml'
  }

  // No parameters block here—JJB YAML defines them

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
          def testFile = params.K6_TEST_FILE ?: ''
          sh "echo '---- Test File ----'"
          sh "echo ${testFile}
          currentBuild.description = "${testFile}"
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

    stage('Run k6 Test') {
      steps {
        dir('dartboard') {
          script {
            // Use the 'sh' step with the 'basename' shell command to securely get the filename.
            testFileBasename = sh(script: "basename ${params.K6_TEST_FILE}", returnStdout: true).trim().replace('.js', '')
            env.K6_SUMMARY_LOG = "${testFileBasename}-k6-summary.log"
            def k6SummaryJsonFile = "${testFileBasename}-summary.json"
            def k6ReportHtmlFile = "${testFileBasename}-report.html"


            // Create the k6 environment file on the agent first.
            // This avoids permission issues inside the container, as the container
            // only needs to read/source the file, not create it.
            def k6EnvContent = """
BASE_URL=${baseURL}
KUBECONFIG=${kubeconfigContainerPath ? kubeconfigContainerPath : ''}
K6_TEST=${params.K6_TEST_FILE}
K6_NO_USAGE_REPORT=true
${params.K6_ENV}
"""
            writeFile file: "./${env.K6_ENV_FILE}", text: k6EnvContent

            sh """
              echo "--- k6.env contents ---"
              cat ${env.K6_ENV_FILE}
              echo "-----------------------"

              docker run --rm --name dartboard-k6-runner \\
                -v "${pwd()}:/app" \\
                --workdir /app \\
                --user=\$(id -u) \\
                --entrypoint='' \\
                ${env.IMAGE_NAME}:latest sh -c '''
                  echo "Sourcing environment and running test..."
                  set -o allexport
                  source "${env.K6_ENV_FILE}"
                  set +o allexport

                  echo "Running k6 test: ${params.K6_TEST_FILE}..."
                  k6 run --no-color ${params.K6_TEST_FILE} | tee ${env.K6_SUMMARY_LOG}
                '''
            """
          }
        }
      }
    }

    stage('Upload Results to S3') {
      steps {
        dir('dartboard') {
          script {
            def s3UploadDir = "k6-results"

            // Determine the S3 path prefix using a ternary operator for conciseness.
            def s3BucketPath = params.DEPLOYMENT_ID ? "${params.DEPLOYMENT_ID}/${env.S3_ARTIFACT_PREFIX}" : env.S3_ARTIFACT_PREFIX

            property.useWithCredentials(['AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY']) {
              sh script: """
                set -x
                echo "Preparing k6 artifacts for S3 upload..."
                mkdir -p ${s3UploadDir}

                # Explicitly copy only the generated k6 report files
                cp -v ./${testFileBasename}-*.json ${s3UploadDir}/ 2>/dev/null || true
                cp -v ./${testFileBasename}-*.xml ${s3UploadDir}/ 2>/dev/null || true
                cp -v ./${testFileBasename}-*.html ${s3UploadDir}/ 2>/dev/null || true
                cp -v ./${env.K6_SUMMARY_LOG} ${s3UploadDir}/ 2>/dev/null || true

                echo "Uploading k6 artifacts from ${s3UploadDir}..."
                docker run --rm \\
                    -v "${pwd()}/${s3UploadDir}:/artifacts" \\
                    -e AWS_ACCESS_KEY_ID \\
                    -e AWS_SECRET_ACCESS_KEY \\
                    -e AWS_S3_REGION="${params.S3_BUCKET_REGION}" \\
                    amazon/aws-cli s3 cp /artifacts/ "s3://${params.S3_BUCKET_NAME}/${s3BucketPath}" --recursive
                rm -rf ${s3UploadDir}
              """, returnStatus: true
            }
          }
        }
      }
    }

    stage('Report to QASE') {
      when { expression { return params.REPORT_TO_QASE } }
      steps {
        dir('dartboard') {
            script {
                def k6SummaryJsonFile = "${testFileBasename}-summary.json"
                def k6ReportHtmlFile = "${testFileBasename}-summary.html"
                withCredentials([string(credentialsId: "QASE_AUTOMATION_TOKEN", variable: "QASE_TESTOPS_API_TOKEN")]) {
                  sh """
                  docker run --rm --name dartboard-qase-reporter \\
                      -v "${pwd()}:/app" \\
                      --workdir /app \\
                      --user=\$(id -u) \\
                      --entrypoint='' \\
                      -e QASE_TESTOPS_API_TOKEN \\
                      -e QASE_TESTOPS_PROJECT="${params.QASE_TESTOPS_PROJECT}" \\
                      -e QASE_TESTOPS_RUN_ID="${params.QASE_TESTOPS_RUN_ID}" \\
                      -e QASE_TEST_RUN_NAME="${params.QASE_TEST_RUN_NAME}" \\
                      -e QASE_TEST_CASE_NAME="${params.QASE_TEST_CASE_NAME}" \\
                      -e K6_SUMMARY_JSON_FILE="./${k6SummaryJsonFile}" \\
                      -e K6_SUMMARY_HTML_FILE="./${k6ReportHtmlFile}" \\
                      ${env.IMAGE_NAME}:latest sh -c '''
                          echo "Reporting k6 results to Qase..."
                          if command -v qasereporter-k6 >/dev/null 2>&1; then
                              source "${env.K6_ENV_FILE}"
                              qasereporter-k6
                          else
                              echo "qasereporter-k6 not found, skipping report."
                              exit 1
                          fi
                      '''
                  """
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
          dartboard/${testFileBasename}-*.json,
          dartboard/${testFileBasename}-*.log,
          dartboard/${testFileBasename}-*.html,
          dartboard/${testFileBasename}-*.xml,
          dartboard/${env.K6_SUMMARY_LOG},
        """.trim(), fingerprint: true

        // The k6 container is run with --rm, so it should clean itself up.
        // But if the job is aborted, the container might be left running.
        echo "Cleaning up Docker resources..."
        try {
          echo "Attempting to remove container: dartboard-k6-runner"
          sh "docker rm -f dartboard-k6-runner"
        } catch (e) {
          echo "Could not remove container 'dartboard-k6-runner'. It may have already been removed. Details: ${e.message}"
        }
        try {
          echo "Attempting to remove container: dartboard-qase-reporter"
          sh "docker rm -f dartboard-qase-reporter"
        } catch (e) {
          echo "Could not remove container 'dartboard-qase-reporter'. It may have already been removed. Details: ${e.message}"
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
