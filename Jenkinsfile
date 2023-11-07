pipeline {
  agent {
    node {
      label 'base'
    }
  }
  environment {
    GITHUB_CREDENTIAL_ID = "github-id"
    DOCKER_REPO_NAMESPACE = 'frp-sigs'
    APP_NAME = 'frp-provisioner/controller-manager'
    DOCKER_REPO_ADDRESS = 'docker.pkg.github.com'
  }
  stages {
    stage('checkout scm') {
      steps {
        checkout(scm)
      }
    }
    stage('build & push snapshot') {
      steps {
        container('base') {
          sh 'podman build -t $DOCKER_REPO_ADDRESS/$DOCKER_REPO_NAMESPACE/$APP_NAME:SNAPSHOT-$BRANCH_NAME-$BUILD_NUMBER .'
          withCredentials([usernamePassword(passwordVariable : 'DOCKER_PASSWORD' ,usernameVariable : 'DOCKER_USERNAME' ,credentialsId : "$GITHUB_CREDENTIAL_ID" ,)]) {
            sh 'echo "$DOCKER_PASSWORD" | podman login  $DOCKER_REPO_ADDRESS -u "$DOCKER_USERNAME" --password-stdin'
            sh 'podman push  $DOCKER_REPO_ADDRESS/$DOCKER_REPO_NAMESPACE/$APP_NAME:SNAPSHOT-$BRANCH_NAME-$BUILD_NUMBER'
          }
        }
      }
    }
    stage('push latest'){
       when{
         branch 'main'
       }
       steps{
         input(id: 'push-as-latest', message: 'Docker re-tag the current image and push it as the latest?')
         container('base'){
           sh 'podman tag  $DOCKER_REPO_ADDRESS/$DOCKER_REPO_NAMESPACE/$APP_NAME:SNAPSHOT-$BRANCH_NAME-$BUILD_NUMBER $DOCKER_REPO_ADDRESS/$DOCKER_REPO_NAMESPACE/$APP_NAME:latest '
           sh 'podman push  $DOCKER_REPO_ADDRESS/$DOCKER_REPO_NAMESPACE/$APP_NAME:latest '
         }
       }
    }
  }
}
