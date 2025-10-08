#!groovy
// Declarative Pipeline Syntax
@Library('qa-jenkins-library') _

def scmWorkspace
def generatedNames
def parseToHTML(text) { text.replace('&', '&amp;').replace('<', '&lt;').replace('>', '&gt;') }

pipeline {
  // agent { label 'vsphere-vpn-1' }
  agent any

  environment {
    // Define environment variables here.  These are available throughout the pipeline.
    imageName = 'dartboard'
    qaseToken = credentials('QASE_AUTOMATION_TOKEN')
    qaseEnvFile = '.qase.env'
    k6EnvFile = 'k6.env'
    k6TestsDir = "k6"
    k6OutputJson = 'k6-output.json'
    k6SummaryLog = 'k6-summary.log'
    harvesterKubeconfig = 'harvester.kubeconfig'
    templateDartFile = 'template-dart.yaml'
    renderedDartFile = 'rendered-dart.yaml'
    envFile = ".env" // Used by container.run
    DEFAULT_PROJECT_NAME = "${JOB_NAME.split('/').last()}-${BUILD_NUMBER}"
    accessDetailsLog = 'access-details.log'
    summaryHtmlFile = 'summary.html'
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
            property.useWithProperties(['AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY']) {
              echo "OUTPUTTING FILE STRUCTURE FOR MANUAL VERIFICATION:"
              sh "ls -al"
              echo "OUTPUTTING ENV FOR MANUAL VERIFICATION:"
              echo "Storing env in file"
              sh "printenv | egrep '^(ARM_|CATTLE_|ADMIN|USER|DO|RANCHER_|AWS_|DEBUG|LOGLEVEL|DEFAULT_|OS_|DOCKER_|CLOUD_|KUBE|BUILD_NUMBER|AZURE|TEST_|QASE_|SLACK_|harvester|K6_TEST|TF_).*=.+' | sort > ${env.envFile}"
              sh "docker build -t ${env.imageName}:latest ."
            }
          }
        }
      }
    }

    stage('Setup SSH Keys') {
      steps {
        script {
          def sshScript = """
            echo "${env.SSH_PEM_KEY}" | base64 -di > ${env.SSH_KEY_NAME}.pem
            chmod 600 ${env.SSH_KEY_NAME}.pem
            chown k6:k6 ${env.SSH_KEY_NAME}.pem
            echo "${env.SSH_PUB_KEY}" > ${env.SSH_KEY_NAME}.pub
            chmod 644 ${env.SSH_KEY_NAME}.pub
            chown k6:k6 ${env.SSH_KEY_NAME}.pub
            echo "VERIFICATION FOR PUB KEY:"
            cat ${env.SSH_KEY_NAME}.pub
            pwd
            ls -al
          """
          generatedNames = generate.names()
          sh """
            docker run --rm --name ${generatedNames.container} \\
              -v ${pwd()}:/home/ \\
              --workdir /home/dartboard \\
              --env-file dartboard/${env.envFile} \\
              --entrypoint='' --user root \\
              ${env.imageName}:latest /bin/sh -c '${sshScript}'
          """
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
            def renderedDart = dartTemplate.replaceAll('\\$\\{HARVESTER_KUBECONFIG\\}', "/home/dartboard/${env.harvesterKubeconfig}")
                                            .replaceAll('\\$\\{SSH_KEY_NAME\\}', "/home/dartboard/${params.SSH_KEY_NAME}")
                                            .replaceAll('\\$\\{PROJECT_NAME\\}', env.DEFAULT_PROJECT_NAME)
            writeFile file: env.renderedDartFile, text: renderedDart

            echo "DUMPING INPUT FILES FOR MANUAL VERIFICATION"
            echo "---- k6.env ----"
            sh "cat ${env.k6EnvFile}"
            echo "---- harvester.kubeconfig ----"
            sh "cat ${env.harvesterKubeconfig}"
            echo "---- renderdDart ----"
            println renderedDart
            echo "---- rendered-dart.yaml ----"
            sh "cat ${env.renderedDartFile}"
          }
        }
      }
    }

    stage('Setup Infrastructure') {
        steps {
          script {
            sh """
              docker run --rm --name ${generatedNames.container} \\
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

    stage('Get Access Details') {
      steps {
        script {
          sh """
            docker run --rm --name ${generatedNames.container} \\
              -v ${pwd()}:/home/ \\
              --workdir /home/dartboard/ \\
              --env-file dartboard/${env.envFile} \\
              --entrypoint='' --user root \\
              ${env.imageName}:latest /bin/sh -c 'dartboard \\
              --dart ${env.renderedDartFile} get-access > ${env.accessDetailsLog}'
          """
          echo "---- Access Details ----"
          sh "cat dartboard/${env.accessDetailsLog}"
        }
      }
    }

    stage('Run Validation Tests') {
        steps {
            script {
              def k6BaseCommand = "k6 run --out json=${env.k6OutputJson} ${env.k6TestsDir}/${params.K6_TEST} | tee ${env.k6SummaryLog}"
              def k6TestCommand = fileExists("${env.k6EnvFile}") ? "set -o allexport; source ${env.k6EnvFile}; set +o allexport; ${k6BaseCommand}" : k6BaseCommand

              sh """
                docker run --rm --name ${generatedNames.container} \\
                  -v ${pwd()}:/home/ \\
                  --workdir /home/dartboard/ \\
                  --env-file dartboard/${env.envFile} \\
                  --entrypoint='' --user root \\
                  ${env.imageName}:latest /bin/sh -c '${k6TestCommand}'
              """
            }
        }
    }

    stage('Generate Build Summary') {
      steps {
        dir('dartboard') {
          script {
            // Create a tarball of the config directory for easy download
            def configDirName = sh(script: "find . -type d -name '*_config' | head -n 1", returnStdout: true).trim()
            if (configDirName) {
              sh "tar -czvf ${configDirName}.tar.gz ${configDirName}"
            }

            // Generate the HTML content
            def htmlContent = """
              <html>
                <head>
                  <title>Build Summary for ${env.JOB_NAME} #${env.BUILD_NUMBER}</title>
                  <style>
                    body { font-family: sans-serif; background-color: #1e1e1e; color: #d4d4d4; }
                    h1, h2 { color: #d4d4d4; }
                    h2 { border-bottom: 1px solid #3c3c3c; padding-bottom: 5px; }
                    pre { background-color: #252526; border: 1px solid #3c3c3c; padding: 10px; white-space: pre-wrap; word-wrap: break-word; color: #ce9178; }
                    ul { list-style-type: none; padding-left: 0; }
                    li { margin-bottom: 10px; }
                    a { color: #3794ff; text-decoration: none; }
                    a:hover { text-decoration: underline; }
                    i { color: #808080; }
                  </style>
                </head>
                <body>
                  <h1>Build Summary: ${env.JOB_NAME} #${env.BUILD_NUMBER}</h1>

                  <h2>k6 Test Summary</h2>
                  <pre>${parseToHTML(readFile(env.k6SummaryLog))}</pre>

                  <h2>Cluster Access Details</h2>
                  <pre>${parseToHTML(readFile(env.accessDetailsLog))}</pre>

                  <h2>Downloads</h2>
                  <ul>
                    ${configDirName ? "<li><a href='${configDirName}.tar.gz'>Download Cluster Configs (${configDirName}.tar.gz)</a></li>" : ""}
                    <li><a href='${env.k6OutputJson}'>Download k6 JSON Output (${env.k6OutputJson})</a></li>
                    <li><a href='${env.renderedDartFile}'>Download Rendered DART File (${env.renderedDartFile})</a></li>
                  </ul>
                  <p><i>See 'Archived Artifacts' for all generated files, including tofu state.</i></p>
                </body>
              </html>
            """
            writeFile file: env.summaryHtmlFile, text: htmlContent
          }
        }
      }
    }
  }
  post {
    always {
      script {
          echo "Archiving Terraform state and K6 test results..."
          // The workspace is shared, so artifacts are on the agent
          archiveArtifacts artifacts: 'dartboard/**/*.tfstate*, dartboard/**/*.json, dartboard/**/*.pem, dartboard/**/*.pub, dartboard/**/*.yaml, dartboard/**/*.sh, dartboard/**/*.env, dartboard/**/*.log, dartboard/**/*.html, dartboard/**/*.tar.gz', fingerprint: true

          // Cleanup Docker image
          try {
            sh "docker rmi -f ${env.imageName}:latest"
          } catch (e) {
            echo "Could not remove docker image ${env.imageName}:latest. It may have already been removed. ${e.message}"
          }

          publishHTML(target: [
            allowMissing: false,
            alwaysLinkToLastBuild: true,
            keepAll: true,
            reportDir: 'dartboard',
            reportFiles: env.summaryHtmlFile,
            reportName: "Build Summary"
          ])
      }
    }
  }
}
