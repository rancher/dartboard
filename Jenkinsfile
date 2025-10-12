#!groovy
// Declarative Pipeline Syntax
@Library('qa-jenkins-library') _

def scmWorkspace
def generatedNames
def configDirPath
def tfstateDir
def parseToHTML(text) { text.replace('&', '&amp;').replace('<', '&lt;').replace('>', '&gt;') }

@NonCPS
def extractRancherUrl(logText) {
    def matcher = (logText =~ /Rancher UI: (https?:\/\/[^\s]+)/)
    if (matcher.find()) {
        return matcher.group(1)
    }
    return null
}

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
    // - https://pkg.go.dev/go.k6.io/k6/errext/exitcodes - 99 = threshold failed
    // exit code 0 = success
    // any other code = Error
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
          def maxRetries = 3
          for (int attempt = 1; attempt <= maxRetries; attempt++) {
            try {
              echo "Attempting to deploy infrastructure... (Attempt ${attempt} of ${maxRetries})"
              sh """
                docker run --rm --name ${generatedNames.container} \\
                  -v ${pwd()}:/home/ \\
                  --workdir /home/dartboard/ \\
                  --env-file dartboard/${env.envFile} \\
                  --entrypoint='' --user root \\
                  ${env.imageName}:latest dartboard \\
                  --dart ${env.renderedDartFile} redeploy
              """
              echo "Infrastructure deployed successfully."
              return // Exit the stage on success
            } catch (e) {
              echo "Attempt ${attempt} failed. Error: ${e.message}"
              if (attempt == maxRetries) {
                echo "All deployment attempts have failed. Running dartboard destroy..."
                try {
                  sh """
                    docker run --rm --name ${generatedNames.container}-destroy \\
                      -v ${pwd()}:/home/ \\
                      --workdir /home/dartboard/ \\
                      --env-file dartboard/${env.envFile} \\
                      --entrypoint='' --user root \\
                      ${env.imageName}:latest dartboard \\
                      --dart ${env.renderedDartFile} destroy
                  """
                } catch (destroyError) {
                  echo "Dartboard destroy command also failed: ${destroyError.message}"
                }
                // Fail the pipeline after cleanup
                error("Infrastructure deployment failed after ${maxRetries} attempts.")
              }
              sleep(15) // Wait for 15 seconds before retrying
            }
          }
        }
      }
      post {
        success {
          script {
            // Dynamically determine the OpenTofu state directory path
            def tofuMainDir = sh(
                script: """docker run --rm \\
                -v ${pwd()}/dartboard:/app \\
                --entrypoint='' --user root \\
                  ${env.imageName}:latest yq '.tofu_main_directory' /app/${env.renderedDartFile}""",
                returnStdout: true
            ).trim().replace('./', '') // Clean up the path

            // Find the generated config directory to create the archive
            configDirPath = sh(script: "find dartboard -type d -name '*_config' | head -n 1", returnStdout: true).trim()

            tfstateDir = "dartboard/${tofuMainDir}/terraform.tfstate.d/"

            if (fileExists(tfstateDir)) {
              echo "Creating OpenTofu state archive from '${tfstateDir}'..."
              def archiveName = "tfstate-${env.DEFAULT_PROJECT_NAME}.zip"
              sh """
                docker run --rm --name ${generatedNames.container}-zip \\
                  -v ${pwd()}/dartboard:/app \\
                  --workdir /app \\
                  --entrypoint='' --user root \\
                  ${env.imageName}:latest sh -c 'cd ${tofuMainDir}/terraform.tfstate.d/ && zip -r ../../../../${archiveName} .'
              """
            } else {
              echo "Could not find OpenTofu state directory at '${tfstateDir}', skipping archive creation."
            }

            if (configDirPath) {
              echo "Creating config archive using Docker..."
              def archiveName = "${configDirPath.split('/').last()}.zip"
              // Create the zip inside the dartboard directory
              sh """
                docker run --rm --name ${generatedNames.container}-zip \\
                  -v ${pwd()}:/home/ \\
                  --workdir /home/ \\
                  --entrypoint='' --user root \\
                  ${env.imageName}:latest zip -r dartboard/${archiveName} ${configDirPath}
              """
            }
          }
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
            if (!configDirPath) {
              error("No `configDirPath` was found.")
            }

            echo "Parsing Rancher URL from access details..."
            def rancherBaseUrl = extractRancherUrl(readFile("dartboard/${env.accessDetailsLog}"))
            if (!rancherBaseUrl) {
                error("Could not find Rancher UI URL in access details log.")
            }
            echo "Found Rancher BASE_URL: ${rancherBaseUrl}"

            def k6BaseCommand = "k6 run --out json=${env.k6OutputJson} ${env.k6TestsDir}/${params.K6_TEST} | tee ${env.k6SummaryLog}"
            // Prepend environment variables for k6. The paths are relative to the container's workdir.
            def k6TestCommand = "export BASE_URL='${rancherBaseUrl}' && export KUBECONFIG='/home/${configDirPath}/upstream.yaml' && export CONTEXT='upstream' && " +
                                (fileExists("${env.k6EnvFile}") ? "set -o allexport; source ${env.k6EnvFile}; set +o allexport; " : "") + k6BaseCommand

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
  }

  post {
    always {
      script {
          echo "Generating build summary..."
          dir('dartboard') {
            def k6Summary = fileExists(env.k6SummaryLog) ? parseToHTML(readFile(env.k6SummaryLog)) : "k6 summary log not found."
            def accessDetails = fileExists(env.accessDetailsLog) ? parseToHTML(readFile(env.accessDetailsLog)) : "Access details log not found."
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
                  <pre>${k6Summary}</pre>

                  <h2>Cluster Access Details</h2>
                  <pre>${accessDetails}</pre>

                  <h2>Downloads</h2>
                  <ul>
                    ${fileExists(env.renderedDartFile) ? "<li><a href='${env.BUILD_URL}artifact/dartboard/${env.renderedDartFile}' download target='_blank'>Download Rendered DART File (${env.renderedDartFile})</a></li>" : ""}
                    ${configDirPath ? "<li><a href='${env.BUILD_URL}artifact/dartboard/${configDirPath.split('/').last()}.zip' download target='_blank'>Download Cluster Configs (${configDirPath.split('/').last()}.zip)</a></li>" : ""}
                    ${fileExists("tfstate-${env.DEFAULT_PROJECT_NAME}.zip") ? "<li><a href='${env.BUILD_URL}artifact/dartboard/tfstate-${env.DEFAULT_PROJECT_NAME}.zip' download target='_blank'>Download OpenTofu State (tfstate-${env.DEFAULT_PROJECT_NAME}.zip)</a></li>" : ""}
                    ${fileExists(env.k6OutputJson) ? "<li><a href='${env.BUILD_URL}artifact/dartboard/${env.k6OutputJson}' download target='_blank'>Download k6 JSON Output (${env.k6OutputJson})</a></li>" : ""}
                  </ul>
                  <p><i>See 'Archived Artifacts' for all generated files, including tofu state.</i></p>
                </body>
              </html>
            """
            writeFile file: env.summaryHtmlFile, text: htmlContent
          }

          echo "Archiving Terraform state and K6 test results..."
          // The workspace is shared, so artifacts are on the agent
          archiveArtifacts artifacts: """
              dartboard/*.html,
              dartboard/*.json,
              dartboard/*.yaml,
              dartboard/*.log,
              dartboard/*.pem,
              dartboard/*.pub,
              dartboard/**/.terraform.tfstate.d/**/*.tfstate,
              dartboard/**/*.tfstate,
              dartboard/**/*.tfstate.backup,
              dartboard/*.zip,
            """.trim(), fingerprint: true

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
