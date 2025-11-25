#!groovy
// Declarative Pipeline Syntax
@Library('qa-jenkins-library') _

def agentLabel = 'jenkins-qa-jenkins-agent'
if (params.HARVESTER_KUBECONFIG) {
    agentLabel = 'vsphere-vpn-1'
}

def scmWorkspace
def generatedNames
def configDirPath
def parseToHTML(text) { text.replace('&', '&amp;').replace('<', '&lt;').replace('>', '&gt;') }

def runningContainerName
def finalSSHKeyName
def finalSSHPemKey
def finalProjectName

pipeline {
  agent { label agentLabel }

  environment {
    // Define environment variables here.  These are available throughout the pipeline.
    imageName = 'dartboard'
    harvesterKubeconfig = 'harvester.kubeconfig'
    templateDartFile = 'template-dart.yaml'
    renderedDartFile = 'rendered-dart.yaml'
    envFile = ".env"
    DEFAULT_PROJECT_NAME = "${JOB_NAME.split('/').last()}-${BUILD_NUMBER}"
    accessDetailsLog = 'access-details.log'
    summaryHtmlFile = 'summary.html'
  }

  // No parameters block here—JJB YAML defines them


  stages {
    stage('Initialize & Checkout') {
      steps {
        script {
          // Use useWithCredentials to securely handle the PEM key
          property.useWithCredentials(['AWS_SSH_PEM_KEY_NAME', 'AWS_SSH_PEM_KEY']) {
            // Initialize variables with fallback logic
            finalSSHPemKey = params.SSH_PEM_KEY ? params.SSH_PEM_KEY : env.AWS_SSH_PEM_KEY
            def sshKeyNameFromCreds = env.AWS_SSH_PEM_KEY_NAME.trim().split('\\.')[0]
            finalSSHKeyName = params.SSH_KEY_NAME ? params.SSH_KEY_NAME : sshKeyNameFromCreds
            sh """
            echo '---- SSH KEY NAME ---'
            echo ${finalSSHKeyName}
            """
            scmWorkspace = project.checkout(repository: params.REPO, branch: params.BRANCH, target: 'dartboard')
          }
        }
      }
    }

    stage('Configure and Build') {
      steps {
        dir('dartboard') {
          script {
            property.useWithProperties(['AWS_ACCESS_KEY_ID', 'AWS_SECRET_ACCESS_KEY']) {
              echo "Storing env in file"
              sh "printenv | egrep '^(ARM_|CATTLE_|ADMIN|USER|DO|RANCHER_|AWS_|DEBUG|LOGLEVEL|DEFAULT_|OS_|DOCKER_|CLOUD_|KUBE|BUILD_NUMBER|AZURE|TEST_|SLACK_|harvester|TF_).*=.+' | sort > ${env.envFile}"
              if (params.EXTRA_ENV_VARS) {
                sh "echo \"${params.EXTRA_ENV_VARS}\" >> ${env.envFile}"
              }
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
              --entrypoint='' \\
              --user=\$(id -u) \\
              ${env.imageName}:latest sleep infinity
          """
        }
      }
    }

    stage('Set Build Description') {
      steps {
        script {
          // Use yq inside the running service container to parse the rancher_version from the DART file contents
          def rancherVersion = sh(
            script: "docker exec ${runningContainerName} sh -c 'echo \"\$1\" | yq .chart_variables.rancher_version' -- '${params.DART_FILE}'",
            returnStdout: true
          ).trim()

          sh "echo '---- Rancher Version ----'"
          sh "echo ${rancherVersion}

          if (rancherVersion && rancherVersion != 'null') {
            currentBuild.description = "Rancher v${rancherVersion}"
          }
        }
      }
    }

    stage('Setup SSH Keys') {
      steps {
        script {
          def sshScript = """
            echo "Writing SSH keys to container..."
            # The PEM key is passed via standard input to avoid issues with special characters
            echo "\${1}" | base64 -d > /dartboard/${finalSSHKeyName}.pem
            chmod 600 /dartboard/${finalSSHKeyName}.pem

            echo "Generating public key from PEM key..."
            ssh-keygen -y -f /dartboard/${finalSSHKeyName}.pem > /dartboard/${finalSSHKeyName}.pub
            chmod 644 /dartboard/${finalSSHKeyName}.pub

            echo "VERIFICATION FOR PUB KEY:"
            cat /dartboard/${finalSSHKeyName}.pub
          """
          sh "docker exec --user=\$(id -u) ${runningContainerName} sh -c '${sshScript}' -- '${finalSSHPemKey}'"
        }
      }
    }

    stage('Determine Project Name') {
      steps {
        script {
          // Use yq to parse the project_name from the DART file contents
          def projectNameFromDart = sh(
            script: "docker exec ${runningContainerName} sh -c 'echo \"\$1\" | yq .tofu_variables.project_name' -- '${params.DART_FILE}'",
            returnStdout: true
          ).trim()

          // Override DEFAULT_PROJECT_NAME if a valid one is found in the DART file
          if (projectNameFromDart && projectNameFromDart != 'null' && !projectNameFromDart.startsWith('$')) {
            echo "Using project_name from DART file: ${projectNameFromDart}"
            finalProjectName = projectNameFromDart
          } else {
            finalProjectName = env.DEFAULT_PROJECT_NAME
            echo "Using default project name: ${env.DEFAULT_PROJECT_NAME}"
          }
        }
      }
    }

    stage('Prepare Parameter Files') {
      steps {
        script {
          property.useWithCredentials(['ADMIN_PASSWORD']) {
            // Render the Dart file using Groovy string replacement
            def dartTemplate = params.DART_FILE
            def renderedDart = dartTemplate.replaceAll('\\$\\{HARVESTER_KUBECONFIG\\}', "/dartboard/${env.harvesterKubeconfig}")
                                            .replaceAll('\\$\\{SSH_KEY_NAME\\}', "/dartboard/${finalSSHKeyName}")
                                            .replaceAll('\\$\\{PROJECT_NAME\\}', env.DEFAULT_PROJECT_NAME)
                                            .replaceAll('\\$\\{ADMIN_PASSWORD\\}', ADMIN_PASSWORD)

            // Use docker exec to write all parameter files to the container
            sh """
              docker exec --user=\$(id -u) --workdir /dartboard ${runningContainerName} sh -c '''
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
    }

    stage('Setup Infrastructure') {
      steps {
        script {
          retry(3) {
            sh """
              docker exec --user=\$(id -u) --workdir /dartboard ${runningContainerName} dartboard \\
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
              docker exec --user=\$(id -u) --workdir /dartboard ${runningContainerName} sh -c '''
                  echo "Creating archives..."
                  tofuMainDir=\$(yq ".tofu_main_directory" ${env.renderedDartFile})
                  tfstateDir="\${tofuMainDir}/terraform.tfstate.d/"
                  configDirPath=\$(find . -type d -name "*_config" | head -n 1)

                  if [ -d "\${tfstateDir}" ]; then
                    echo "Creating OpenTofu state archive from '\${tfstateDir}'..."
                    archiveName="tfstate-${finalProjectName}.zip"
                    (cd "\${tfstateDir}" && zip -r "/dartboard/\${archiveName}" "${finalProjectName}")
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
              docker exec --user=\$(id -u) --workdir /dartboard ${runningContainerName} dartboard \\
              --dart ${env.renderedDartFile} destroy
            """
          }
        }
      }
    }

    stage('Get Access Details') {
      steps {
        script {
          sh "docker exec --user=\$(id -u) --workdir /dartboard ${runningContainerName} sh -c 'dartboard --dart ${env.renderedDartFile} get-access > ${env.accessDetailsLog}'"
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
                  ${fileExists("tfstate-${finalProjectName}.zip") ? "<li><a href='${env.BUILD_URL}artifact/dartboard/tfstate-${finalProjectName}.zip' download target='_blank'>Download OpenTofu State (tfstate-${finalProjectName}.zip)</a></li>" : ""}
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
              cp tfstate-${finalProjectName}.zip ${s3ArtifactsDir}/ 2>/dev/null || true
              cp ${env.accessDetailsLog} ${s3ArtifactsDir}/ 2>/dev/null || true
              if [ -n "${configZipFile}" ]; then cp ${configZipFile} ${s3ArtifactsDir}/ 2>/dev/null || true; fi
            """

            // Run the aws-cli container to upload the files
            sh script: """
              docker run --rm \\
                -v "${pwd()}/${s3ArtifactsDir}:/artifacts" \\
                -e AWS_ACCESS_KEY_ID \\
                -e AWS_SECRET_ACCESS_KEY \\
                -e AWS_S3_REGION="${params.S3_BUCKET_REGION}" \\
                amazon/aws-cli s3 cp /artifacts "s3://${params.S3_BUCKET_NAME}/${finalProjectName}/" --recursive
            """, returnStatus: true

            // Clean up the temporary directory
            sh "rm -rf ${s3ArtifactsDir}"
          }
        }

        echo "Archiving build artifacts..."
        archiveArtifacts artifacts: """
            dartboard/*.html,
            dartboard/*.json,
            dartboard/**/rendered-dart.yaml,
            dartboard/*.log,
            dartboard/*.zip
        """.trim(), fingerprint: true

        // Cleanup Docker resources with explicit logging
        try {
          if (runningContainerName) {
            echo "Attempting to remove service container: ${runningContainerName}"
            sh "docker rm -f ${runningContainerName}"
          }
        } catch (e) {
          echo "Could not remove container '${runningContainerName}'. It may have already been removed or never started. Details: ${e.message}"
        }
        try {
          echo "Attempting to remove image: ${env.imageName}:latest"
          sh "docker rmi -f ${env.imageName}:latest"
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
        sh """
          set -x
          echo "Removing large source and cache directories..."
          rm -rf charts/ docs/ internal/ k6/ tofu/ cmd/ scripts/ darts/

          echo "Removing other non-artifact files..."
          find . -maxdepth 1 -type f \\
            -not -name '*.html' -not -name '*.json' -not -name '*.log' -not -name '*.zip' \\
            -not -name 'rendered-dart.yaml' -not -name 'Jenkinsfile' -delete
        """
      }
    }
  }
}
