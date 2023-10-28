pipeline {
  agent {
    node {
      label 'base'
    }
  }
  environment {
    DOCKER_REPO_NAMESPACE = 'aapelismith'
    APP_NAME = 'frp-provisioner'
    DOCKER_REPO_ADDRESS = 'harbor.devops.kubesphere.local:30280'
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
        }
      }
    }
  }
}
