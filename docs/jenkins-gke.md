# Jenkins ile Google Cloud Deploy

Bu proje için önerilen akış:

1. Jenkins repoyu çeker.
2. Go testlerini çalıştırır.
3. Docker imajını build eder.
4. İmajı Google Artifact Registry'ye push eder.
5. GKE cluster'a Kubernetes manifestlerini uygular.
6. Deployment image tag'ini Jenkins build numarasıyla günceller.
7. Rollout durumunu bekler.
8. Servis bilgisini yazdırarak public IP'yi Jenkins logunda gösterir.

## Jenkins Gereksinimleri

Jenkins agent üzerinde şu araçlar kurulu olmalı:

- `go`
- `docker`
- `gcloud`
- `kubectl`

Jenkins kullanıcısının Docker daemon'a erişimi olmalı.

Jenkins'i çalıştıran Linux kullanıcısı Google Cloud'a giriş yapmış olmalı:

```bash
gcloud auth login
gcloud config set project project-444d504d-38fb-4e0d-83e
gcloud auth configure-docker europe-west1-docker.pkg.dev
```

Not: Repoda `Jenkinsfile` bulunması tek başına otomasyonun aktif olduğu anlamına gelmez. Jenkins üzerinde `Pipeline from SCM` job kurulmalı ve otomatik tetikleme için GitHub webhook veya SCM polling açılmalıdır.

## Google Cloud Hazırlığı

Artifact Registry repo:

```bash
gcloud artifacts repositories create dork-repo \
  --repository-format=docker \
  --location=europe-west1
```

GKE cluster örneği:

```bash
gcloud container clusters create-auto dork-cluster \
  --zone=europe-west1-b
```

Bu projede servis hesabı JSON key dosyası kullanılmaz. Google Cloud organizasyon politikasında `iam.disableServiceAccountKeyCreation` aktifse JSON key üretilemez. Bunun yerine Jenkins, makinedeki aktif `gcloud` kullanıcı oturumu ile deploy eder.

Jenkins'i çalıştıran Google kullanıcısı için gerekli roller:

- Artifact Registry Writer
- Kubernetes Engine Developer
- Service Account User

## Jenkins Credentials

Jenkins > Manage Credentials altında şu credential'ları oluştur:

```text
postgres-user             Secret text
postgres-password         Secret text
postgres-db               Secret text
```

Örnek değerler:

```text
postgres-user=dorkuser
postgres-password=guclu-bir-sifre
postgres-db=dorkdb
```

## Jenkinsfile Ayarları

`Jenkinsfile` içindeki şu değerleri kendi Google Cloud ortamına göre değiştir:

```groovy
GCP_PROJECT_ID = 'project-444d504d-38fb-4e0d-83e'
GCP_REGION = 'europe-west1'
ARTIFACT_REPOSITORY = 'dork-repo'
GKE_CLUSTER = 'dork-cluster'
GKE_LOCATION = 'europe-west1-b'
```

## İlk Deploy

Jenkins job olarak `Pipeline from SCM` seç:

- SCM: Git
- Repository URL: kendi repo adresin
- Script Path: `Jenkinsfile`

Otomatik deploy için job ayarlarında GitHub webhook veya `Poll SCM` tetikleyicisi açılmalıdır. Aksi durumda pipeline dosyası hazırdır ama build manuel başlatılır.

İlk başarılı build sonrası dış IP:

```bash
kubectl get service dork-backend-service
```

## Local Hızlı Kontrol

DB olmadan UI/API kontrolü:

```bash
docker build -t projectofdork-local:check .
docker run --rm -p 9872:9867 projectofdork-local:check
```

Ardından:

```text
http://localhost:9872
```
