def githubCredentialsId = 'managed-pipeline-ci-robot'                       // The GitHub credentials ID for accessing the repo
def githubRepoUrl = 'https://github.com/Indellient/vault-helper.git'        // The GitHUb repo URL for vault-helper
def bldrs = [
    [ url: "https://bldr.bluepipeline.io", origin: "bluepipeline", credentialsId: "depot-token" ],
    [ url: "https://bldr.habitat.sh", origin: "indellient", credentialsId: "public-depot-token" ],
]

pipeline {
    agent none

    /**
     * We handle checkout of SCM ourselves
     */
    options {
        skipDefaultCheckout(true)
    }

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
            steps {
                script {
                    /**
                     * Delete the current workspace files
                     */
                    deleteDir()

                    /**
                     * Perform SCM checkout
                     */
                    git branch: "${env.BRANCH_NAME}", credentialsId: "${githubCredentialsId}", url: "${githubRepoUrl}"

                    /**
                     * Do all builds
                     */
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
                             * Upload & Promote if on master
                             */
                            if (env.BRANCH_NAME == 'master') {
                                /**
                                 * Upload
                                 */
                                habitat task: 'upload', lastBuildFile: "${workspace}/results/last_build.env", authToken: "${HAB_AUTH_TOKEN}", bldrUrl: "${bldrs[i].url}"

                                /**
                                 * Promote
                                 */
                                habitat task: 'promote', channel: 'stable', lastBuildFile: "${workspace}/results/last_build.env", authToken: "${HAB_AUTH_TOKEN}", bldrUrl: "${bldrs[i].url}"

                                /**
                                 * Create GH Release if origin is indellient
                                 */
                                if (bldrs[i].origin == 'indellient') {
                                    withCredentials([usernamePassword(credentialsId: "${githubCredentialsId}", usernameVariable: 'GITHUB_USER', passwordVariable: 'GITHUB_TOKEN')]) {
                                        withEnv(["GITHUB_REPO=${githubRepoUrl}"]) {
                                            sh 'ls -lah *'
                                            // Create the release
                                            sh '. results/last_build.env && bin/gothub release --user Indellient --security-token ${GITHUB_TOKEN} --repo $( basename "${GITHUB_REPO}" | sed "s/.git//g" ) --tag $pkg_name-$pkg_version-$pkg_release --name "$pkg_name v$pkg_version-$pkg_release"'

                                            // Upload the files (no .hart files)
                                            sh '. results/last_build.env && bin/gothub upload --user Indellient --security-token ${GITHUB_TOKEN} --repo $( basename "${GITHUB_REPO}" | sed "s/.git//g" ) --tag $pkg_name-$pkg_version-$pkg_release --name vault-helper-linux-amd64 --file bin/vault-helper-linux-amd64'
                                            sh '. results/last_build.env && bin/gothub upload --user Indellient --security-token ${GITHUB_TOKEN} --repo $( basename "${GITHUB_REPO}" | sed "s/.git//g" ) --tag $pkg_name-$pkg_version-$pkg_release --name vault-helper-windows-amd64.exe --file bin/vault-helper-windows-amd64.exe'
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
