pipeline {
    agent {
        docker {
            image 'golang:1.20.4-bullseye'
            args '-u root'
        }
    }

    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }
        stage('Install dependencies') {
            steps {
                sh 'go mod download && go mod verify'
            }
        }
        stage('Build') {
            steps {
                sh 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ./pocketbase'
            }
        }
        stage('Copy artifacts to VPS') {
            steps {
                sshPublisher(
                    continueOnError: false, 
                    failOnError: true,
                    publishers: [
                        sshPublisherDesc(
                            configName: "elykp.com",
                            transfers: [sshTransfer(
                                sourceFiles: 'pocketbase', 
                                remoteDirectory: "elykp-api", 
                                cleanRemote: false, 
                                )
                            ],
                            verbose: true,
                        )
                    ]
                )
            }
        }
    }

    post {
        failure {
            emailext to: "blackparadise0407@gmail.com",
            subject: "jenkins build:${currentBuild.currentResult}: ${env.JOB_NAME}",
            body: "${currentBuild.currentResult}: Job ${env.JOB_NAME}\nMore Info can be found here: ${env.BUILD_URL}"
        }
    }
}