#!groovy
// Declarative Pipeline Syntax
@Library('qa-jenkins-library') _

pipeline {
  agent { label params.JENKINS_AGENT_LABEL }

  environment {
    IMAGE_NAME = 'dartboard'
    K6_ENV_FILE = 'k6.env'
    K6_SUMMARY_LOG = 'k6-summary.log'
    S3_ARTIFACT_PREFIX = "${JOB_NAME.split('/').last()}-${BUILD_NUMBER}"
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

    stage('Run k6 Test') {
      steps {
        dir('dartboard') {
          script {
            sh """
              docker run --rm --name dartboard-k6-runner \\
                -v "${pwd()}:/app" \\
                --workdir /app \\
                --entrypoint='' \\
                ${env.IMAGE_NAME}:latest sh -c '''
                  echo "Writing k6 environment variables..."
                  echo "${params.K6_ENV}" > ${env.K6_ENV_FILE}

                  echo "Sourcing environment and running test..."
                  set -o allexport && source "${env.K6_ENV_FILE}" && set +o allexport

                  echo "Running k6 test..."
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
                  amazon/aws-cli s3 cp /artifacts/ "s3://${params.S3_BUCKET_NAME}/${env.S3_ARTIFACT_PREFIX}/" \\
                  --recursive --include "k6-output.json" --include "k6-summary.log"
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
              archiveArtifacts artifacts: "dartboard/*.json, dartboard/*.log", fingerprint: true
          }
      }
  }
}
