// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

def DOCKER_IMAGE_TAG
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
        DOCKER_CREDS = credentials('github-packages-credentials-rw')
        DOCKER_REPO = 'ghcr.io'
        DOCKER_NAMESPACE = 'verrazzano'
        NETRC_FILE = credentials('netrc')
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

                script {
                    def props = readProperties file: '.verrazzano-development-version'
                    VERRAZZANO_DEV_VERSION = props['verrazzano-development-version']
                    TIMESTAMP = sh(returnStdout: true, script: "date +%Y%m%d%H%M%S").trim()
                    SHORT_COMMIT_HASH = sh(returnStdout: true, script: "git rev-parse --short HEAD").trim()
                    DOCKER_IMAGE_TAG = "${VERRAZZANO_DEV_VERSION}-${TIMESTAMP}-${SHORT_COMMIT_HASH}"
                }
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
                    make push DOCKER_REPO=${env.DOCKER_REPO} DOCKER_NAMESPACE=${env.DOCKER_NAMESPACE} DOCKER_IMAGE_NAME=${DOCKER_IMAGE_NAME} DOCKER_IMAGE_TAG=${DOCKER_IMAGE_TAG} CREATE_LATEST_TAG=${CREATE_LATEST_TAG}
                   """
            }
        }

        stage('gofmt Check') {
            when { not { buildingTag() } }
            steps {
                sh """
                    cd ${GO_REPO_PATH}/verrazzano-operator
                    make go-fmt
                """
            }
        }

        stage('go vet Check') {
            when { not { buildingTag() } }
            steps {
                sh """
                    cd ${GO_REPO_PATH}/verrazzano-operator
                    make go-vet
                """
            }
        }

        stage('golint Check') {
            when { not { buildingTag() } }
            steps {
                sh """
                    cd ${GO_REPO_PATH}/verrazzano-operator
                    make go-lint
                """
            }
        }

        stage('ineffassign Check') {
            when { not { buildingTag() } }
            steps {
                sh """
                    cd ${GO_REPO_PATH}/verrazzano-operator
                    make go-ineffassign
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
                    cp coverage.xml ${WORKSPACE}
                    build/scripts/copy-junit-output.sh ${WORKSPACE}
                """
            }
            post {
                always {
                    archiveArtifacts artifacts: '**/coverage.html', allowEmptyArchive: true
                    cobertura(coberturaReportFile: 'coverage.xml',
                      enableNewApi: true,
                      autoUpdateHealth: false,
                      autoUpdateStability: false,
                      failUnstable: true,
                      failUnhealthy: true,
                      failNoReports: true,
                      onlyStable: false,
                      conditionalCoverageTargets: '50, 0, 0',
                      fileCoverageTargets: '50, 0, 0',
                      lineCoverageTargets: '50, 0, 0',
                      packageCoverageTargets: '80, 0, 0',
                    )
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
                    clairScanTemp "${env.DOCKER_REPO}/${env.DOCKER_NAMESPACE}/${DOCKER_IMAGE_NAME}:${DOCKER_IMAGE_TAG}"
                }
            }
            post {
                always {
                    archiveArtifacts artifacts: '**/scanning-report.json', allowEmptyArchive: true
                }
            }
        }

        stage('Integration Tests') {
            when { not { buildingTag() } }
            steps {
                sh """
                    cd ${GO_REPO_PATH}/verrazzano-operator
                    make integ-test DOCKER_IMAGE_NAME=${DOCKER_IMAGE_NAME} DOCKER_IMAGE_TAG=${DOCKER_IMAGE_TAG}
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
    }

    post {
        failure {
            mail to: "${env.BUILD_NOTIFICATION_TO_EMAIL}", from: "${env.BUILD_NOTIFICATION_FROM_EMAIL}",
            subject: "Verrazzano: ${env.JOB_NAME} - Failed",
            body: "Job Failed - \"${env.JOB_NAME}\" build: ${env.BUILD_NUMBER}\n\nView the log at:\n ${env.BUILD_URL}\n\nBlue Ocean:\n${env.RUN_DISPLAY_URL}"
        }
    }
}

