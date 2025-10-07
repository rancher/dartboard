#!groovy
// Declarative Pipeline Syntax
@Library('qa-jenkins-library') _

def scmWorkspace

pipeline {
  // agent { label 'vsphere-vpn-1' }
  agent any

  environment {
    // Define environment variables here.  These are available throughout the pipeline.
    imageName = 'dartboard'
    qaseToken = credentials('QASE_AUTOMATION_TOKEN')
    qaseEnvFile = '.qase.env'
    k6EnvFile = 'k6.env'
    k6TestsDir = "k6/"
    k6OutputJson = 'k6-output.json'
    k6SummaryLog = 'k6-summary.log'
    harvesterKubeconfig = 'harvester.kubeconfig'
    templateDartFile = 'template-dart.yaml'
    renderedDartFile = 'rendered-dart.yaml'
    envFile = ".env" // Used by container.run
    DEFAULT_PROJECT_NAME = "${JOB_NAME.split('/').last()}-${BUILD_NUMBER}"
  }

  // No parameters block here—JJB YAML defines them

  stages {
    stage('Checkout') {
        steps {
          script {
            scmWorkspace = project.checkout(repository: params.REPO, branch: params.BRANCH, target: 'dartboard')
          }
        }
    }

    // TODO: Set up a QASE client to utilize these for logging test run results + artifacts
    stage('Create QASE Environment Variables') {
        steps {
            script {
                def qase = 'REPORT_TO_QASE=' + params.REPORT_TO_QASE + '\n' +
                            'QASE_PROJECT_ID=' + params.QASE_PROJECT_ID + '\n' +
                            'QASE_TEST_RUN_ID=' + params.QASE_TEST_RUN_ID + '\n' +
                            'QASE_TEST_RUN_NAME=' + params.QASE_TEST_CASE_ID + '\n' +
                            'QASE_AUTOMATION_TOKEN=' + qaseToken + '\n' // Use credentials plugin
                writeFile file: qaseEnvFile, text: qase
                sh """
                set -o allexport
                echo '---- .qase.env ----'
                source ${qaseEnvFile}
                printenv
                set +o allexport
                """
            }
        }
    }

    stage('Configure and Build') {
        steps {
          dir('dartboard') {
            script {
              echo "OUTPUTTING FILE STRUCTURE FOR MANUAL VERIFICATION:"
              sh "ls -al"
              echo "OUTPUTTING ENV FOR MANUAL VERIFICATION:"
              echo "Storing env in file"
              sh "printenv | egrep '^(ARM_|CATTLE_|ADMIN|USER|DO|RANCHER_|AWS_|DEBUG|LOGLEVEL|DEFAULT_|OS_|DOCKER_|CLOUD_|KUBE|BUILD_NUMBER|AZURE|TEST_|QASE_|SLACK_|harvester|K6_TEST|TF_).*=.+' | sort > ${env.envFile}"
              sh "echo 'TF_LOG=DEBUG' >> ${env.envFile}"
              sh "docker build -t ${env.imageName}:latest ."
            }
          }
        }
    }

    stage('Prepare Parameter Files') {
      steps {
        dir('dartboard'){
          script {
            // Write supporting files from parameters
            writeFile file: env.k6EnvFile, text: params.K6_ENV
            writeFile file: env.harvesterKubeconfig, text: params.HARVESTER_KUBECONFIG

            // Render the Dart file using Groovy string replacement instead of envsubst
            def dartTemplate = params.DART_FILE
            def renderedDart = dartTemplate.replace('$HARVESTER_KUBECONFIG', env.harvesterKubeconfig)
                                           .replace('$SSH_KEY_NAME', params.SSH_KEY_NAME)
                                           .replace('$PROJECT_NAME', env.DEFAULT_PROJECT_NAME)

            writeFile file: env.renderedDartFile, text: renderedDart

            echo "DUMPING INPUT FILES FOR MANUAL VERIFICATION"
            echo "---- k6.env ----"
            sh "cat ${env.k6EnvFile}"
            echo "---- harvester.kubeconfig ----"
            sh "cat ${env.harvesterKubeconfig}"
            echo "---- rendered-dart.yaml ----"
            sh "cat ${env.renderedDartFile}"
          }
        }
      }
    }

    stage('Setup Infrastructure') {
        steps {
          script {
            def names = generate.names()
            sh """
              docker run --rm --name ${names.container} \\
                -v ${pwd()}:/home/ \\
                --workdir /home/dartboard/ \\
                --env-file dartboard/${env.envFile} \\
                --entrypoint='' --user root \\
                ${env.imageName}:latest dartboard \\
                --dart ${env.renderedDartFile} deploy
            """
          }
        }
    }

    stage('Run Validation Tests') {
        steps {
            script {
              def k6BaseCommand = "k6 run --out json=${env.k6OutputJson} ${env.k6TestsDir}/${params.K6_TEST} | tee ${env.k6SummaryLog}"
              def k6TestCommand = fileExists("${env.k6EnvFile}") ? "set -o allexport; source ${env.k6EnvFile}; set +o allexport; ${k6BaseCommand}" : k6BaseCommand

              def names = generate.names()
              sh """
                docker run --rm --name ${names.container} \\
                  -v ${pwd()}:/home/ \\
                  --workdir /home/dartboard/ \\
                  --env-file dartboard/${env.envFile} \\
                  --entrypoint='' --user root \\
                  ${env.imageName}:latest /bin/sh -c '${k6TestCommand}'
              """
            }
        }
    }
  }
  post {
    always {
      script {
          echo "Archiving Terraform state and K6 test results..."
          // The workspace is shared, so artifacts are on the agent
          archiveArtifacts artifacts: 'dartboard/**/*.tfstate*, dartboard/**/*.json, dartboard/**/*.pem, dartboard/**/*.pub, dartboard/**/*.yaml, dartboard/**/*.sh, dartboard/**/*.env', fingerprint: true

          // Cleanup Docker image
          try {
            container.remove([ [name: jobContainer.name, image: jobContainer.image] ])
          } catch (e) {
            echo "Could not remove docker image ${env.imageName}:latest. It may have already been removed. ${e.message}"
          }
      }
    }
  }
}
