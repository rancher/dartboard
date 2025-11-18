#!groovy
// Declarative Pipeline Syntax
@Library('qa-jenkins-library') _

def agentLabel = 'jenkins-qa-jenkins-agent'
if (params.JENKINS_AGENT_LABEL) {
  agentLabel = params.JENKINS_AGENT_LABEL
}

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
    // These will be populated in the 'Prepare Environment' stage
    RANCHER_FQDN        = ''
    KUBECONFIG_PATH     = ''
    // Base URL for the k6 test
    BASE_URL            = ''
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

    stage('Build Dartboard Image') {
      steps {
        dir('dartboard') {
          sh "docker build -t ${env.IMAGE_NAME}:latest ."
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

                echo "Downloaded artifacts:"
                ls -l ${env.ARTIFACTS_DIR}

                # Unzip the config archive
                config_zip=\$(find ${env.ARTIFACTS_DIR} -name '*_config.zip' | head -n 1)
                if [ -n "\$config_zip" ]; then
                  unzip -o "\$config_zip" -d "${env.ARTIFACTS_DIR}"
                else
                  echo "Warning: No config zip file found."
                fi
              """
            }
            // Extract FQDN and set environment variables for the next stage
            sh "mv ${env.ARTIFACTS_DIR}/${env.ACCESS_LOG} ./${env.ACCESS_LOG}"
            def accessLogPath = "./${env.ACCESS_LOG}"
            if (fileExists(accessLogPath)) {
              def accessLogContent = readFile(accessLogPath)
              // See https://docs.groovy-lang.org/next/html/groovy-jdk/java/util/regex/Matcher.html
              def matcher = accessLogContent =~ /(?m)^Rancher UI:\s*(https?:\/\/[^ :]+)/
              if (matcher.find()) {
                def match = matcher.group(1).trim()
                env.BASE_URL = "https://${match}"
                echo "Found Rancher URL: ${env.BASE_URL}"
                sh "rm ${accessLogPath}"
              } else {
                echo "Warning: Could not find 'Rancher UI' in ${env.ACCESS_LOG}"
              }
            }

            sh "mv ${env.ARTIFACTS_DIR}/${env.KUBECONFIG_FILE} ./${env.KUBECONFIG_FILE}"
            def kubeconfigPath = "./${env.KUBECONFIG_FILE}"
            if (fileExists(kubeconfigPath)) {
              // Absolute path relative to the container's filespace
              env.KUBECONFIG_PATH = "/app/${env.KUBECONFIG_FILE}"
              echo "Found kubeconfig at: ${env.KUBECONFIG_PATH}"
            }
          }
        }
      }
    }

    stage('Run k6 Test') {
      steps {
        dir('dartboard') {
          script {
            // Create the k6 environment file on the agent first.
            // This avoids permission issues inside the container, as the container
            // only needs to read/source the file, not create it.
            def k6EnvContent = """
BASE_URL=${env.BASE_URL}
KUBECONFIG=${env.KUBECONFIG_PATH ? env.KUBECONFIG_PATH : ''}
K6_TEST=${params.K6_TEST_FILE}
${params.K6_ENV}
"""
            writeFile file: env.K6_ENV_FILE, text: k6EnvContent

            sh """
              echo "--- k6.env contents ---"
              cat ${env.K6_ENV_FILE}
              echo "-----------------------"

              docker run --rm --name dartboard-k6-runner \\
                -v "${pwd()}:/app" \\
                --workdir /app \\
                --user "\$(id -u):\$(id -g)" \\
                --entrypoint='' \\
                ${env.IMAGE_NAME}:latest sh -c '''
                  echo "Sourcing environment and running test..."
                  set -o allexport
                  source "${env.K6_ENV_FILE}"
                  set +o allexport

                  echo "Running k6 test: ${params.K6_TEST_FILE}..."
                  k6 run ${params.K6_TEST_FILE} | tee ${env.K6_SUMMARY_LOG}
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
            property.useWithCredentials(['AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY']) {
              sh script: """
              docker run --rm \\
                  -v "${pwd()}:/artifacts" \\
                  -e AWS_ACCESS_KEY_ID \\
                  -e AWS_SECRET_ACCESS_KEY \\
                  -e AWS_S3_REGION="${params.S3_BUCKET_REGION}" \\
                  amazon/aws-cli s3 cp /artifacts/ "s3://${params.S3_BUCKET_NAME}/${params.DEPLOYMENT_ID ?: env.S3_ARTIFACT_PREFIX}/k6/" --recursive \\
                  --exclude "*" \\
                  --exclude "charts/*" \\
                  --include "*.json" \\
                  --include "*.log" \\
                  --include "*.xml" \\
                  --include "*.html"
              """, returnStatus: true
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
