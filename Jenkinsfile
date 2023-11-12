pipeline {
  agent {
    node {
      label 'go21'
    }
  }
  environment {
    GOPROXY = "https://goproxy.cn,direct"
    GITHUB_CREDENTIAL_ID = "github-id"
    KUBECONFIG_CREDENTIAL_ID = "kubeconfig-id"
    DOCKER_REPO_NAMESPACE = "frp-sigs"
    APP_NAME = "frp-provisioner/controller-manager"
    DOCKER_REPO_ADDRESS = "docker.pkg.github.com"
  }
  stages {
    stage('checkout scm') {
      steps {
        checkout(scm)
      }
    }
    stage('build & push snapshot') {
      steps {
        container('go') {
          sh 'IMG=$DOCKER_REPO_ADDRESS/$DOCKER_REPO_NAMESPACE/$APP_NAME:SNAPSHOT-$BRANCH_NAME-$BUILD_NUMBER make deploy'
          sh 'podman build -t $DOCKER_REPO_ADDRESS/$DOCKER_REPO_NAMESPACE/$APP_NAME:SNAPSHOT-$BRANCH_NAME-$BUILD_NUMBER .'
          withCredentials([usernamePassword(passwordVariable : 'DOCKER_PASSWORD' ,usernameVariable : 'DOCKER_USERNAME' ,credentialsId : "$GITHUB_CREDENTIAL_ID" ,)]) {
            sh 'echo "$DOCKER_PASSWORD" | podman login  $DOCKER_REPO_ADDRESS -u "$DOCKER_USERNAME" --password-stdin'
            sh 'podman push  $DOCKER_REPO_ADDRESS/$DOCKER_REPO_NAMESPACE/$APP_NAME:SNAPSHOT-$BRANCH_NAME-$BUILD_NUMBER'
          }
        }
      }
    }
    stage('deploy to dev') {
      steps {
        kubernetesDeploy(configs: 'deploy/**', enableConfigSubstitution: true, kubeconfigId: "$KUBECONFIG_CREDENTIAL_ID")
      }
    }
    stage('push latest'){
       when{
         branch 'main'
       }
       steps{
         input(id: 'push-as-latest', message: 'Docker re-tag the current image and push it as the latest?')
         container('go'){
           sh 'podman tag  $DOCKER_REPO_ADDRESS/$DOCKER_REPO_NAMESPACE/$APP_NAME:SNAPSHOT-$BRANCH_NAME-$BUILD_NUMBER $DOCKER_REPO_ADDRESS/$DOCKER_REPO_NAMESPACE/$APP_NAME:latest '
           sh 'podman push  $DOCKER_REPO_ADDRESS/$DOCKER_REPO_NAMESPACE/$APP_NAME:latest '
         }
       }
    }
  }
}
