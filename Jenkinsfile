// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

def HEAD_COMMIT
def skipBuild = false 

pipeline {
    options {
        skipDefaultCheckout true
        disableConcurrentBuilds()
    }

    agent {
       docker {
            image "${RUNNER_DOCKER_IMAGE}"
            args "${RUNNER_DOCKER_ARGS}"
            registryUrl "${RUNNER_DOCKER_REGISTRY_URL}"
            registryCredentialsId 'ocir-pull-and-push-account'
        }
    }

    parameters {
        string (name: 'RELEASE_VERSION',
                defaultValue: '',
                description: 'Release version used for the version of helm chart and tag for the image:\n'+
                'When RELEASE_VERSION is not defined, version will be determined by incrementing last minor release version by 1, for example:\n'+
                'When RELEASE_VERSION is v0.1.0, image tag will be v0.1.0 and helm chart version is also v0.1.0.\n'+
                'When RELEASE_VERSION is not specified and last release version is v0.1.0, image tag will be v0.1.1 and helm chart version is also v0.1.1.',
                trim: true)
        string (name: 'RELEASE_DESCRIPTION',
                defaultValue: '',
                description: 'Brief description for the release.',
                trim: true)
        string (name: 'RELEASE_BRANCH',
                defaultValue: 'master',
                description: 'Branch to create release from, change this to enable release from a non master branch, e.g.\n'+
                'When the branch being built is master then release will always be created when RELEASE_BRANCH has the default value - master.\n'+
                'When the branch being built is any non-master branch - release can be created by setting RELEASE_BRANCH to same value as non-master branch, else it is skipped.\n',
                trim: true)
    }

    environment {
        DOCKER_CI_IMAGE_NAME = 'verrazzano-operator-jenkins'
        DOCKER_PUBLISH_IMAGE_NAME = 'verrazzano-operator'
        DOCKER_IMAGE_NAME = "${env.BRANCH_NAME == 'master' ? env.DOCKER_PUBLISH_IMAGE_NAME : env.DOCKER_CI_IMAGE_NAME}"
        CREATE_LATEST_TAG = "${env.BRANCH_NAME == 'master' ? '1' : '0'}"
        GOPATH = '/home/opc/go'
        GO_REPO_PATH = "${GOPATH}/src/github.com/verrazzano"
        DOCKER_CREDS = credentials('ocir-pull-and-push-account')
        NETRC_FILE = credentials('netrc')
        OCI_CLI_TENANCY = credentials('oci-tenancy')
        OCI_CLI_USER = credentials('oci-user-ocid')
        OCI_CLI_FINGERPRINT = credentials('oci-api-key-fingerprint')
        OCI_CLI_KEY_FILE = credentials('oci-api-key')
        OCI_CLI_REGION = 'us-phoenix-1'
        GITHUB_API_TOKEN = credentials('github-api-token-release-assets')
        GITHUB_RELEASE_USERID = credentials('github-userid-release')
        GITHUB_RELEASE_EMAIL = credentials('github-email-release')
    }

    stages {
        stage('Clean workspace and checkout') {
            steps {
                sh "rm -rf github.com/verrazzano/verrazzano-coh-cluster-operator"
                sh "rm -rf $GOPATH/pkg/mod/github.com/verrazzano/verrazzano-coh-cluster-operator"
                script {
                    checkout scm
                    result = sh (script: "git log -1 | grep '.*\\[automatic helm release\\].*'", returnStatus: true) 
                    if (result == 0) {
                        echo ("'automatic helm release' spotted in git commit. No further stages will be executed.")
                        skipBuild = true
                        currentBuild.description = "[automatic helm release] Build Skipped."
                        sh "exit 0"
                    }
                }
                sh """
                    cp -f "${NETRC_FILE}" $HOME/.netrc
                    chmod 600 $HOME/.netrc
                """

                sh """
                    echo "${DOCKER_CREDS_PSW}" | docker login ${env.DOCKER_REPO} -u ${DOCKER_CREDS_USR} --password-stdin
                    rm -rf ${GO_REPO_PATH}/verrazzano-operator
                    mkdir -p ${GO_REPO_PATH}/verrazzano-operator
                    tar cf - . | (cd ${GO_REPO_PATH}/verrazzano-operator/ ; tar xf -)
                """
            }
        }

        stage('Build') {
            when {
                allOf {
                    not { buildingTag() }
                    equals expected: false, actual: skipBuild
                }
            }
            steps {
                sh """
                    cd ${GO_REPO_PATH}/verrazzano-operator
                    make push DOCKER_REPO=${env.DOCKER_REPO} DOCKER_NAMESPACE=${env.DOCKER_NAMESPACE} DOCKER_IMAGE_NAME=${DOCKER_IMAGE_NAME} CREATE_LATEST_TAG=${CREATE_LATEST_TAG}
                   """
            }
        }

        stage('Third Party License Check') {
            when {
                allOf {
                    not { buildingTag() }
                    equals expected: false, actual: skipBuild
                }
            }
            steps {
                thirdpartyCheck()
            }
        }

        stage('Copyright Compliance Check') {
            when {
                allOf {
                    not { buildingTag() }
                    equals expected: false, actual: skipBuild
                }
            }
            steps {
                copyrightScan "${WORKSPACE}"
            }
        }

        stage('Unit Tests') {
            when {
                allOf {
                    not { buildingTag() }
                    equals expected: false, actual: skipBuild
                }
            }
            steps {
                sh """
                    cd ${GO_REPO_PATH}/verrazzano-operator
                    make unit-test
                    make -B coverage
                    cp coverage.html ${WORKSPACE}
                    build/scripts/copy-junit-output.sh ${WORKSPACE} 
                """
            }
            post {
                always {
                    archiveArtifacts artifacts: '**/coverage.html', allowEmptyArchive: true
                    junit testResults: '**/*test-result.xml', allowEmptyResults: true
                }
            }
        }

        stage('Scan Image') {
            when {
                allOf {
                    not { buildingTag() }
                    equals expected: false, actual: skipBuild
                }
            }
            steps {
                script {
                    HEAD_COMMIT = sh(returnStdout: true, script: "git rev-parse HEAD").trim()
                    clairScanTemp "${env.DOCKER_REPO}/${env.DOCKER_NAMESPACE}/${DOCKER_IMAGE_NAME}:${HEAD_COMMIT}"
                }
            }
            post {
                always {
                    archiveArtifacts artifacts: '**/scanning-report.json', allowEmptyArchive: true
                }
            }
        }

        stage('Publish Image') {
            when { buildingTag() }
            steps {
                sh """
                    cd ${GO_REPO_PATH}/verrazzano-operator
                    make push-tag DOCKER_REPO=${env.DOCKER_REPO} DOCKER_NAMESPACE=${env.DOCKER_NAMESPACE} DOCKER_IMAGE_NAME=${env.DOCKER_IMAGE_NAME} RELEASE_VERSION=${params.RELEASE_VERSION} RELEASE_DESCRIPTION=${params.RELEASE_DESCRIPTION} RELEASE_BRANCH=${params.RELEASE_BRANCH}
                """
            }
        }
    }

    post {
        failure {
            mail to: "${env.BUILD_NOTIFICATION_TO_EMAIL}", from: "${env.BUILD_NOTIFICATION_FROM_EMAIL}",
            subject: "Verrazzano: ${env.JOB_NAME} - Failed",
            body: "Job Failed - \"${env.JOB_NAME}\" build: ${env.BUILD_NUMBER}\n\nView the log at:\n ${env.BUILD_URL}\n\nBlue Ocean:\n${env.RUN_DISPLAY_URL}"
        }
    }
}
