/**
 * This Jenkinsfile is meant to build the vault-helper binaries using go's built-in cross compile functionality, and
 * upload to GitHub releases page only.
 */
def githubCredentialsId = 'managed-pipeline-ci-robot'                       // The GitHub credentials ID for accessing the repo
def githubRepoUrl = 'https://github.com/Indellient/vault-helper.git'        // The GitHUb repo URL for vault-helper
def bldrUrl = 'https://bldr.habitat.sh'                                     // Builder URL
def bldrOrigin = 'indellient'                                               // Builder Origin
def bldrCredentialsId = 'public-depot-token'                                // Builder Credentials

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
        stage('Build') {
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

                    withEnv(["GITHUB_REPO=${githubRepoUrl}"]) {
                        withCredentials([
                                string(credentialsId: "${bldrCredentialsId}", variable: 'HAB_AUTH_TOKEN'),
                                usernamePassword(credentialsId: "${githubCredentialsId}", usernameVariable: 'GITHUB_USER', passwordVariable: 'GITHUB_TOKEN')
                        ]) {
                            /**
                             * Download Keys
                             */
                            sh "hab origin key download ${bldrOrigin} --auth ${HAB_AUTH_TOKEN} --url ${bldrUrl}"
                            sh "hab origin key download ${bldrOrigin} --auth ${HAB_AUTH_TOKEN} --url ${bldrUrl} --secret"

                            /**
                             * Build
                             */
                            dir("${workspace}") {
                                habitat task: 'build', directory: '.', origin: "${bldrOrigin}", bldrUrl: "${bldrUrl}", docker: true
                            }

                            /**
                             * Generate Checksums
                             */
                            sh 'sha256sum bin/vault-helper-linux-amd64 > bin/vault-helper-linux-amd64.sha256'
                            sh 'sha256sum bin/vault-helper-windows-amd64.exe > bin/vault-helper-windows-amd64.exe.sha256'
                        }
                    }
                }
            }
        }

        /**
         * Create GH Release & Tag
         */
        stage('Release') {
            agent {
                node {
                    label 'lnx'
                }
            }

            steps {
                script {
                    /**
                     * Only run the GH release if we don't have a matching tag from the pkg_version and we are on master branch
                     */
                    if (env.BRANCH_NAME == "master" && sh(returnStdout: true, script: '. results/last_build.env && git tag --list v$pkg_version').trim() == "") {
                        // Create the release
                        sh '. results/last_build.env && bin/gothub release --user Indellient --security-token ${GITHUB_TOKEN} --repo $( basename "${GITHUB_REPO}" | sed "s/.git//g" ) --tag v$pkg_version --name "v$pkg_version"'

                        // Upload the files (no .hart files)
                        sh '. results/last_build.env && bin/gothub upload --user Indellient --security-token ${GITHUB_TOKEN} --repo $( basename "${GITHUB_REPO}" | sed "s/.git//g" ) --tag v$pkg_version --name vault-helper-linux-amd64 --file bin/vault-helper-linux-amd64'
                        sh '. results/last_build.env && bin/gothub upload --user Indellient --security-token ${GITHUB_TOKEN} --repo $( basename "${GITHUB_REPO}" | sed "s/.git//g" ) --tag v$pkg_version --name vault-helper-linux-amd64.sha256 --file bin/vault-helper-linux-amd64.sha256'
                        sh '. results/last_build.env && bin/gothub upload --user Indellient --security-token ${GITHUB_TOKEN} --repo $( basename "${GITHUB_REPO}" | sed "s/.git//g" ) --tag v$pkg_version --name vault-helper-windows-amd64.exe --file bin/vault-helper-windows-amd64.exe'
                        sh '. results/last_build.env && bin/gothub upload --user Indellient --security-token ${GITHUB_TOKEN} --repo $( basename "${GITHUB_REPO}" | sed "s/.git//g" ) --tag v$pkg_version --name vault-helper-windows-amd64.exe.sha256 --file bin/vault-helper-windows-amd64.exe.sha256'
                    }
                }
            }
        }
    }
}
