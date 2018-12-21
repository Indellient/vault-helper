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
                withCredentials([string(credentialsId: 'depot-token', variable: 'HAB_AUTH_TOKEN')]) {
                    sh "hab origin key download bluepipeline --auth ${HAB_AUTH_TOKEN} --url ${env.HAB_BLDR_URL}"
                    sh "hab origin key download bluepipeline --auth ${HAB_AUTH_TOKEN} --url ${env.HAB_BLDR_URL} --secret"
                }

                dir("${workspace}") {
                    habitat task: 'build', directory: '.', origin: env.HAB_ORIGIN, bldrUrl: env.HAB_BLDR_URL
                }

                withCredentials([string(credentialsId: 'depot-token', variable: 'HAB_AUTH_TOKEN')]) {
                    habitat task: 'upload', lastBuildFile: "${workspace}/results/last_build.env", authToken: env.HAB_AUTH_TOKEN, bldrUrl: env.HAB_BLDR_URL
                }

                dir("${workspace}") {
                    sh 'hab studio rm'
                }
            }
        }
    }
}
