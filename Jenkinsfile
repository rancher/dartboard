#!groovy
// Declarative Pipeline Syntax
@Library('qa-jenkins-library') _

def scmWorkspace
def generatedNames
def configDirPath
def parseToHTML(text) { text.replace('&', '&amp;').replace('<', '&lt;').replace('>', '&gt;') }

def runningContainerName

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
    harvesterKubeconfig = 'harvester.kubeconfig'
    templateDartFile = 'template-dart.yaml'
    renderedDartFile = 'rendered-dart.yaml'
    envFile = ".env" // Used by container.run
    DEFAULT_PROJECT_NAME = "${JOB_NAME.split('/').last()}-${BUILD_NUMBER}"
    accessDetailsLog = 'access-details.log'
    summaryHtmlFile = 'summary.html'
    s3BucketName='jenkins-bullseye-storage'
    s3BucketRegion='us-east-2'
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
              sh "printenv | egrep '^(ARM_|CATTLE_|ADMIN|USER|DO|RANCHER_|AWS_|DEBUG|LOGLEVEL|DEFAULT_|OS_|DOCKER_|CLOUD_|KUBE|BUILD_NUMBER|AZURE|TEST_|QASE_|SLACK_|harvester|TF_).*=.+' | sort > ${env.envFile}"
              sh "docker build -t ${env.imageName}:latest ."
            }
          }
        }
      }
    }

    stage('Start Service Container') {
      steps {
        script {
          generatedNames = generate.names()
          runningContainerName = "${generatedNames.container}-service"
          sh """
            docker run -d --rm --name ${runningContainerName} \\
              -v ${pwd()}/dartboard:/dartboard \\
              --workdir /dartboard \\
              --env-file dartboard/${env.envFile} \\
              --entrypoint='' --user root \\
              ${env.imageName}:latest sleep infinity
          """
        }
      }
    }

    stage('Setup SSH Keys') {
      steps {
        script {
          def sshScript = """
            echo "Writing SSH keys to container..."
            echo "${env.SSH_PEM_KEY}" | base64 -d > /dartboard/${env.SSH_KEY_NAME}.pem
            chmod 600 /dartboard/${env.SSH_KEY_NAME}.pem
            echo "${env.SSH_PUB_KEY}" > /dartboard/${env.SSH_KEY_NAME}.pub
            chmod 644 /dartboard/${env.SSH_KEY_NAME}.pub
            echo "VERIFICATION FOR PUB KEY:"
            cat /dartboard/${env.SSH_KEY_NAME}.pub
            pwd
            ls -al
          """
          sh "docker exec --user root ${runningContainerName} sh -c '${sshScript}'"
        }
      }
    }

    stage('Prepare Parameter Files') {
      steps {
        script {
          // Render the Dart file using Groovy string replacement
          def dartTemplate = params.DART_FILE
          def renderedDart = dartTemplate.replaceAll('\\$\\{HARVESTER_KUBECONFIG\\}', "/dartboard/${env.harvesterKubeconfig}")
                                          .replaceAll('\\$\\{SSH_KEY_NAME\\}', "/dartboard/${params.SSH_KEY_NAME}")
                                          .replaceAll('\\$\\{PROJECT_NAME\\}', env.DEFAULT_PROJECT_NAME)

          // Use docker exec to write all parameter files to the container
          sh """
            docker exec --user root --workdir /dartboard ${runningContainerName} sh -c '''
              echo "Writing parameter files to container using here-documents to preserve special characters..."

              # Write HARVESTER_KUBECONFIG to harvester.kubeconfig
              cat <<'EOF' > ${env.harvesterKubeconfig}
${params.HARVESTER_KUBECONFIG}
EOF
              # Write the rendered DART file
              cat <<'EOF' > ${env.renderedDartFile}
${renderedDart}
EOF
              echo "DUMPING INPUT FILES FOR MANUAL VERIFICATION"
              echo "---- harvester.kubeconfig ----"
              cat ${env.harvesterKubeconfig}
              echo "---- rendered-dart.yaml ----"
              cat ${env.renderedDartFile}
            '''
          """
        }
      }
    }

    stage('Setup Infrastructure') {
      steps {
        script {
          retry(3) {
            sh """
              docker exec --user root --workdir /dartboard ${runningContainerName} dartboard \\
                --dart ${env.renderedDartFile} redeploy
            """
          }
        }
      }
      post {
        success {
          script {
            // Dynamically determine the OpenTofu state directory path
            // Create archives inside the container
            sh """
              docker exec --user root --workdir /dartboard ${runningContainerName} sh -c '''
                  echo "Creating archives..."
                  pwd
                  ls -al
                  tofuMainDir=\$(yq ".tofu_main_directory" ${env.renderedDartFile})
                  tfstateDir="\${tofuMainDir}/terraform.tfstate.d/"
                  configDirPath=\$(find . -type d -name "*_config" | head -n 1)

                  if [ -d "\${tfstateDir}" ]; then
                    echo "Creating OpenTofu state archive from '\${tfstateDir}'..."
                    archiveName="tfstate-${env.DEFAULT_PROJECT_NAME}.zip"
                    (cd "\${tfstateDir}" && zip -r "/dartboard/\${archiveName}" "${env.DEFAULT_PROJECT_NAME}")
                  else
                    echo "Could not find OpenTofu state directory at '\${tfstateDir}', skipping archive creation."
                  fi

                  if [ -n "\${configDirPath}" ] && [ -d "\${configDirPath}" ]; then
                    echo "Creating config archive from \${configDirPath}..."
                    archiveName="\$(basename \${configDirPath}).zip"
                    (cd \${configDirPath} && zip -r "/dartboard/\${archiveName}" ./)
                  fi
              '''
            """
          }
        }
        failure {
          script {
            echo "Setup failed. Running dartboard destroy..."
            sh """
              docker exec --user root --workdir /dartboard ${runningContainerName} dartboard \\
              --dart ${env.renderedDartFile} destroy
            """
          }
        }
      }
    }

    stage('Get Access Details') {
      steps {
        script {
          sh "docker exec --user root --workdir /dartboard ${runningContainerName} sh -c 'dartboard --dart ${env.renderedDartFile} get-access > ${env.accessDetailsLog}'"
          echo "---- Access Details ----"
          sh "docker exec ${runningContainerName} cat /dartboard/${env.accessDetailsLog}"
        }
      }
    }

  }

  post {
    always {
      script {
        echo "Generating build summary..."
        // Artifacts are on the agent workspace via the volume mount, no copy needed.

        dir('dartboard') {
          def accessDetails = fileExists(env.accessDetailsLog) ? parseToHTML(readFile(env.accessDetailsLog)) : "Access details log not found."

          // We need to find the config dir path again on the agent for the summary link
          try {
              configDirPath = sh(script: "find . -type d -name '*_config' | head -n 1", returnStdout: true).trim()
          } catch (e) {
              configDirPath = null
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

                <h2>Cluster Access Details</h2>
                <pre>${accessDetails}</pre>

                <h2>Downloads</h2>
                <ul>
                  ${fileExists(env.renderedDartFile) ? "<li><a href='${env.BUILD_URL}artifact/dartboard/${env.renderedDartFile}' download target='_blank'>Download Rendered DART File (${env.renderedDartFile})</a></li>" : ""}
                  ${configDirPath ? "<li><a href='${env.BUILD_URL}artifact/dartboard/${configDirPath.split('/').last()}.zip' download target='_blank'>Download Cluster Configs (${configDirPath.split('/').last()}.zip)</a></li>" : ""}
                  ${fileExists("tfstate-${env.DEFAULT_PROJECT_NAME}.zip") ? "<li><a href='${env.BUILD_URL}artifact/dartboard/tfstate-${env.DEFAULT_PROJECT_NAME}.zip' download target='_blank'>Download OpenTofu State (tfstate-${env.DEFAULT_PROJECT_NAME}.zip)</a></li>" : ""}
                </ul>
                <p><i>See 'Archived Artifacts' for all generated files, including tofu state.</i></p>
              </body>
            </html>
          """
          writeFile file: env.summaryHtmlFile, text: htmlContent

          property.useWithCredentials(['AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY']) {
            echo "Uploading build artifacts to S3..."
            // Copy files from the workspace (which is mounted into the container) to the temp s3 dir
            def s3ArtifactsDir = "s3-upload-artifacts"
            sh "mkdir -p ${s3ArtifactsDir}"
            // Find the config zip file name first
            def configZipFile = sh(script: "find . -maxdepth 1 -name '*_config.zip' -exec basename {} \\;", returnStdout: true).trim()

            // Explicitly copy only the artifacts used for the build summary and S3 upload
            sh """
              cp ${env.renderedDartFile} ${s3ArtifactsDir}/ 2>/dev/null || true
              cp tfstate-${env.DEFAULT_PROJECT_NAME}.zip ${s3ArtifactsDir}/ 2>/dev/null || true
              if [ -n "${configZipFile}" ]; then cp ${configZipFile} ${s3ArtifactsDir}/ 2>/dev/null || true; fi
            """

            // Run the aws-cli container to upload the files
            sh script: """
              docker run --rm \\
                -v "${pwd()}/${s3ArtifactsDir}:/artifacts" \\
                -e AWS_ACCESS_KEY_ID \\
                -e AWS_SECRET_ACCESS_KEY \\
                -e AWS_S3_REGION="${env.s3BucketRegion}" \\
                amazon/aws-cli s3 cp /artifacts "s3://${env.s3BucketName}/${env.DEFAULT_PROJECT_NAME}/" --recursive
            """, returnStatus: true

            // Clean up the temporary directory
            sh "rm -rf ${s3ArtifactsDir}"
          }
        }

        // Cleanup Docker container and image
        try {
          if (runningContainerName) {
            sh "docker stop ${runningContainerName}"
          }
        } catch (e) {
          echo "Could not stop docker container ${runningContainerName}. It may have already been stopped. ${e.message}"
        }
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
