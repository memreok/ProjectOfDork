pipeline {
    agent any

    options {
        timestamps()
        disableConcurrentBuilds()
        buildDiscarder(logRotator(numToKeepStr: '20'))
    }

    environment {
        GCP_PROJECT_ID = 'project-444d504d-38fb-4e0d-83e'
        GCP_REGION = 'europe-west1'
        ARTIFACT_REPOSITORY = 'dork-repo'
        IMAGE_NAME = 'projectofdork-app'
        GKE_CLUSTER = 'dork-cluster'
        GKE_LOCATION = 'europe-west1-b'
        DEPLOYMENT_NAME = 'dork-backend'
        CONTAINER_NAME = 'dork-backend-container'
        IMAGE_URI = "${env.GCP_REGION}-docker.pkg.dev/${env.GCP_PROJECT_ID}/${env.ARTIFACT_REPOSITORY}/${env.IMAGE_NAME}"
        GOCACHE = "${env.WORKSPACE}/.cache/go-build"
    }

    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Prepare') {
            steps {
                script {
                    env.SHORT_COMMIT = sh(returnStdout: true, script: 'git rev-parse --short HEAD').trim()
                    env.IMAGE_TAG = "${env.BUILD_NUMBER}-${env.SHORT_COMMIT}"
                }
                echo "Image: ${IMAGE_URI}:${IMAGE_TAG}"
            }
        }

        stage('Go Test') {
            steps {
                dir('src/backend') {
                    sh 'go test ./...'
                }
            }
        }

        stage('Docker Build') {
            steps {
                sh '''
                    docker build \
                      -t "${IMAGE_URI}:${IMAGE_TAG}" \
                      -t "${IMAGE_URI}:latest" \
                      .
                '''
            }
        }

        stage('Google Cloud Auth') {
            steps {
                sh '''
                    gcloud config set project "${GCP_PROJECT_ID}"
                    gcloud auth list
                    gcloud auth configure-docker "${GCP_REGION}-docker.pkg.dev" --quiet
                '''
            }
        }

        stage('Docker Push') {
            steps {
                sh '''
                    docker push "${IMAGE_URI}:${IMAGE_TAG}"
                    docker push "${IMAGE_URI}:latest"
                '''
            }
        }

        stage('Deploy to GKE') {
            steps {
                withCredentials([
                    string(credentialsId: 'postgres-user', variable: 'POSTGRES_USER'),
                    string(credentialsId: 'postgres-password', variable: 'POSTGRES_PASSWORD'),
                    string(credentialsId: 'postgres-db', variable: 'POSTGRES_DB')
                ]) {
                    sh '''
                        gcloud container clusters get-credentials "${GKE_CLUSTER}" \
                          --location "${GKE_LOCATION}" \
                          --project "${GCP_PROJECT_ID}"

                        kubectl create secret generic postgres-secret \
                          --from-literal=POSTGRES_USER="${POSTGRES_USER}" \
                          --from-literal=POSTGRES_PASSWORD="${POSTGRES_PASSWORD}" \
                          --from-literal=POSTGRES_DB="${POSTGRES_DB}" \
                          --from-literal=DATABASE_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres-service:5432/${POSTGRES_DB}?sslmode=disable" \
                          --dry-run=client -o yaml | kubectl apply -f -

                        kubectl apply -f k8s/postgres-pvc.yaml
                        kubectl apply -f k8s/postgres-deployment.yaml
                        kubectl apply -f k8s/postgres-service.yaml
                        kubectl apply -f k8s/backend-deployment.yaml
                        kubectl apply -f k8s/backend-service.yaml
                        kubectl apply -f k8s/hpa.yaml
                        kubectl apply -f k8s/network-policy.yaml

                        kubectl set image deployment/${DEPLOYMENT_NAME} \
                          ${CONTAINER_NAME}="${IMAGE_URI}:${IMAGE_TAG}"

                        kubectl rollout status deployment/${DEPLOYMENT_NAME} --timeout=180s
                        kubectl rollout history deployment/${DEPLOYMENT_NAME}
                        kubectl get service dork-backend-service
                    '''
                }
            }
        }
    }

    post {
        success {
            echo "Deploy tamamlandı: ${IMAGE_URI}:${IMAGE_TAG}"
        }
        failure {
            echo 'Pipeline başarısız oldu. Jenkins loglarındaki son hata satırlarını kontrol et.'
        }
    }
}
