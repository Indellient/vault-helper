pipeline {

    agent none

    environment {
        HAB_NOCOLORING = true
        HAB_ORIGIN = 'bluepipeline'
        HAB_BLDR_URL = 'https://bldr.bluepipeline.io'
    }

    stages {
        stage('Build and Upload') {
            agent {
                node {
                    label 'lnx'
                }
            }
            when {
                allOf {
                    branch 'master'
                }
            }
            steps {
                withEnv(["VAULT_HELPER_BUILD_VERSION=1.0.${env.BUILD_NUMBER}"]) {
                    dir("${workspace}/vault-helper") {
                        habitat task: 'build', directory: '.', origin: env.HAB_ORIGIN, bldrUrl: env.HAB_BLDR_URL
                    }
                }
                withCredentials([string(credentialsId: 'depot-token', variable: 'HAB_AUTH_TOKEN')]) {
                    habitat task: 'upload', lastBuildFile: "${workspace}/vault-helper/results/last_build.env", authToken: env.HAB_AUTH_TOKEN, bldrUrl: env.HAB_BLDR_URL
                }
                dir("${workspace}/vault-helper") {
                    sh 'hab studio rm'
                }
            }
        }
    }
}
