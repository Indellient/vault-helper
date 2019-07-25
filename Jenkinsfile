def bldrs = [
    [ url: "https://bldr.bluepipeline.io", origin: "bluepipeline", credentialsId: "depot-token" ],
    [ url: "https://bldr.habitat.sh", origin: "indellient", credentialsId: "public-depot-token" ],
]

pipeline {
    agent none

    environment {
        HAB_NOCOLORING = true
        HAB_NONINTERACTIVE = true
    }

    stages {
        stage('Build, Upload, and Promote') {
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
                script {
                    for (int i=0; i < bldrs.size(); i++) {
                        withCredentials([string(credentialsId: "${bldrs[i].credentialsId}", variable: 'HAB_AUTH_TOKEN')]){
                            /**
                             * Download Keys
                             */
                            sh "hab origin key download ${bldrs[i].origin} --auth ${HAB_AUTH_TOKEN} --url ${bldrs[i].url}"
                            sh "hab origin key download ${bldrs[i].origin} --auth ${HAB_AUTH_TOKEN} --url ${bldrs[i].url} --secret"

                            /**
                             * Build
                             */
                            dir("${workspace}") {
                                habitat task: 'build', directory: '.', origin: "${bldrs[i].origin}", bldrUrl: "${bldrs[i].url}", docker: true
                            }

                            /**
                             * Upload
                             */
                            habitat task: 'upload', lastBuildFile: "${workspace}/results/last_build.env", authToken: "${HAB_AUTH_TOKEN}", bldrUrl: "${bldrs[i].url}"

                            /**
                             * Promote
                             */
                            habitat task: 'promote', channel: 'stable', lastBuildFile: "${workspace}/results/last_build.env", authToken: "${HAB_AUTH_TOKEN}", bldrUrl: "${bldrs[i].url}"
                        }
                    }
                }
            }
        }
    }
}
