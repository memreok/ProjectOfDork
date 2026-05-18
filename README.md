# Dork Atolyesi - Bulut Bilisim Final Projesi

Dork Atolyesi, Go ile gelistirilmis basit bir web uygulamasidir. Kullanici bir domain girer, uygulama hedef domain icin Google Dork sorgulari uretir, hedefin HTTP/HTTPS durumunu kontrol eder ve arama gecmisini PostgreSQL uzerinde saklar.

Bu repo; Docker imaji, Kubernetes manifestleri, Persistent Volume/PVC, NetworkPolicy, HorizontalPodAutoscaler ve Jenkins tabanli CI/CD pipeline dosyalarini icerir. Kubernetes ortami olarak Google Kubernetes Engine (GKE), imaj deposu olarak Google Artifact Registry hedeflenmistir.

## Teslim Checklist

| Ister | Durum | Dosya |
| --- | --- | --- |
| Web uygulamasi | Var | `src/backend`, `src/frontend` |
| Dockerfile | Var | `Dockerfile` |
| Kubernetes Deployment | Var | `k8s/backend-deployment.yaml`, `k8s/postgres-deployment.yaml` |
| Kubernetes Service | Var | `k8s/backend-service.yaml`, `k8s/postgres-service.yaml` |
| Scaling | Var | `k8s/hpa.yaml` |
| Rolling update | Var | `k8s/backend-deployment.yaml`, `Jenkinsfile` |
| Rollback adimlari | Var | README komutlari |
| Persistent Volume/PVC | Var | `k8s/postgres-pvc.yaml` |
| NetworkPolicy | Var | `k8s/network-policy.yaml` |
| CI/CD pipeline | Var | `Jenkinsfile` |
| GKE deploy akisi | Var | `docs/jenkins-gke.md`, `Jenkinsfile` |

## Uygulama Mimarisi

Uygulama iki ana parcadan olusur:

- Go backend: HTTP server, API endpointleri, health/readiness endpointleri ve PostgreSQL baglantisini yonetir.
- HTML frontend: Kullanici formu, dork sonuclari ve gecmis sorgu ekranlarini sunar.

Temel endpointler:

| Endpoint | Gorev |
| --- | --- |
| `/` | Web arayuzu |
| `/api/dorks?domain=example.com` | JSON formatinda dork uretimi |
| `/history` | Kayitli sorgu gecmisi |
| `/health` | Liveness probe |
| `/ready` | Readiness probe, veritabani hazirligini kontrol eder |

PostgreSQL, tarama gecmisini kalici olarak saklamak icin kullanilir. Uygulama Kubernetes icinde `DATABASE_URL` degerini `postgres-secret` Secret nesnesinden alir.

## Sistem Mimarisi

Sistemin bulut ortami su sekildedir:

```text
Kullanici
   |
   v
GKE LoadBalancer Service
   |
   v
dork-backend Deployment (Go web uygulamasi, 2+ pod)
   |
   v
postgres-service (ClusterIP)
   |
   v
postgres-db Deployment
   |
   v
postgres-pvc (kalici disk)
```

Dis dunyaya yalnizca `dork-backend-service` acilir. PostgreSQL servisi `ClusterIP` oldugu icin cluster disindan erisilemez.

## Kubernetes Mimarisi

Kubernetes kaynaklari `k8s/` dizinindedir:

| Kaynak | Aciklama |
| --- | --- |
| `backend-deployment.yaml` | Go uygulamasini calistiran Deployment. RollingUpdate stratejisi, readiness/liveness probe ve resource limitleri icerir. |
| `backend-service.yaml` | Uygulamayi GKE LoadBalancer ile internete acar. |
| `postgres-deployment.yaml` | PostgreSQL containerini calistirir. |
| `postgres-service.yaml` | PostgreSQL icin sadece cluster ici erisim saglar. |
| `postgres-pvc.yaml` | PostgreSQL verilerini kalici disk uzerinde saklar. |
| `postgres-secret.example.yaml` | Ornek PostgreSQL Secret dosyasi. Gercek sifreler repo icine yazilmaz. |
| `admin-secret.example.yaml` | Ornek admin token Secret dosyasi. Gecmis silme islemleri icin kullanilir. |
| `hpa.yaml` | Backend podlarini CPU kullanimina gore otomatik olceklendirir. |
| `network-policy.yaml` | PostgreSQL'e sadece backend podlarindan erisim verir; backend icin gerekli ingress/egress trafigini tanimlar. |

## Docker

Imaj cok asamali build ile uretilir:

1. `golang:1.26.2-alpine` imaji icinde Go binary derlenir.
2. `alpine:latest` runtime imajina yalnizca binary ve frontend dosyalari kopyalanir.
3. Uygulama container icinde `9867` portundan calisir.

Yerel build:

```bash
docker build -t projectofdork-local:latest .
```

Yerel calistirma:

```bash
docker run --rm -p 9867:9867 \
  -e DATABASE_URL="postgres://dorkuser:password@host.docker.internal:5432/dorkdb?sslmode=disable" \
  projectofdork-local:latest
```

Veritabani olmadan sadece UI/API kontrolu icin `DATABASE_URL` verilmeden de calistirilabilir. Kubernetes ortaminda `DB_REQUIRED=true` oldugu icin veritabani hazir degilse pod ready olmaz.

## CI/CD Pipeline Akisi

CI/CD icin Jenkins kullanilir. `Jenkinsfile` su asamalari calistirir:

1. Repository checkout edilir.
2. Commit SHA ve Jenkins build numarasindan imaj tag'i uretilir.
3. `go test ./...` calistirilir.
4. Docker imaji build edilir.
5. Jenkins makinesindeki aktif `gcloud` oturumu ile Google Cloud projesi secilir ve Docker auth yapilir.
6. Imaj Google Artifact Registry'ye push edilir.
7. GKE cluster credentials alinir.
8. Secret, PVC, Deployment, Service, HPA ve NetworkPolicy manifestleri uygulanir.
9. Deployment imaji yeni tag ile guncellenir.
10. `kubectl rollout status` ile rolling update sonucu beklenir.
11. `kubectl get service dork-backend-service` ile public IP Jenkins loguna yazdirilir.

Jenkinsfile'in repoda bulunmasi tek basina otomasyonun aktif oldugu anlamina gelmez. Otomasyonun gercekten calismasi icin Jenkins tarafinda su kurulumlar yapilmis olmalidir:

- Jenkins job tipi `Pipeline from SCM` olmali.
- Repository URL bu GitHub reposunu gostermeli.
- Script Path `Jenkinsfile` olmali.
- Jenkins credential kayitlari olusturulmali.
- Jenkins'i calistiran kullanici daha once `gcloud auth login` ile Google Cloud'a giris yapmis olmali.
- Build tetikleyici olarak GitHub webhook veya belirli araliklarla SCM polling acilmali.

Webhook/polling yoksa Jenkinsfile dogrudur ama build elle baslatilir; tam otomatik sayilmaz.

## Jenkins Credentials

Jenkins > Manage Credentials altinda su credential ID'leri beklenir:

```text
postgres-user             Secret text
postgres-password         Secret text
postgres-db               Secret text
```

Bu projede servis hesabi JSON key kullanilmamaktadir. Google Cloud organizasyon politikasinda `iam.disableServiceAccountKeyCreation` aktif oldugu icin Jenkins, makinedeki aktif `gcloud` kullanici oturumuyla calisir.

Jenkins'i calistiran Google kullanicisinda en az su yetkiler bulunmalidir:

- Artifact Registry Writer
- Kubernetes Engine Developer
- Service Account User

Detayli Jenkins kurulumu icin: `docs/jenkins-gke.md`

## Public IP Bulma

Bu projede public IP, `dork-backend-service` LoadBalancer servisinden gelir.

Cloud Shell veya `gcloud`/`kubectl` kurulu bir makinede:

```bash
gcloud container clusters get-credentials dork-cluster \
  --location europe-west1-b \
  --project project-444d504d-38fb-4e0d-83e

kubectl get service dork-backend-service
```

Sadece IP'yi almak icin:

```bash
kubectl get service dork-backend-service \
  -o jsonpath='{.status.loadBalancer.ingress[0].ip}'
```

Eger `EXTERNAL-IP` kisminda `<pending>` gorunuyorsa LoadBalancer henuz IP almamistir. Biraz bekleyip tekrar kontrol edilmelidir.

## Kubernetes Deploy

Ilk deploy icin Secret olusturulur:

```bash
kubectl create secret generic postgres-secret \
  --from-literal=POSTGRES_USER=dorkuser \
  --from-literal=POSTGRES_PASSWORD='guclu-bir-sifre' \
  --from-literal=POSTGRES_DB=dorkdb \
  --from-literal=DATABASE_URL='postgres://dorkuser:guclu-bir-sifre@postgres-service:5432/dorkdb?sslmode=disable'
```

Normal kullanicilar yalnizca kendi tarayici cookie oturumlarina ait gecmisi gorur ve silebilir. Adminin tum kullanici gecmisini gorebilmesi ve toplu yonetebilmesi icin admin token Secret'i olusturulur. Bu Secret yoksa uygulama calismaya devam eder, ancak admin modu kapali kalir:

```bash
kubectl create secret generic admin-secret \
  --from-literal=ADMIN_TOKEN='uzun-rastgele-bir-token'
```

Manifestleri uygulama:

```bash
kubectl apply -f k8s/postgres-pvc.yaml
kubectl apply -f k8s/postgres-deployment.yaml
kubectl apply -f k8s/postgres-service.yaml
kubectl apply -f k8s/backend-deployment.yaml
kubectl apply -f k8s/backend-service.yaml
kubectl apply -f k8s/hpa.yaml
kubectl apply -f k8s/network-policy.yaml
```

Durum kontrolu:

```bash
kubectl get pods
kubectl get services
kubectl get pvc
kubectl get hpa
kubectl get networkpolicy
```

## Deployment ve Service Kullanimi

Backend Deployment, Go uygulamasini birden fazla pod olarak calistirir. Readiness probe `/ready` endpointini kullanir; veritabani hazir degilse pod trafige alinmaz. Liveness probe `/health` endpointini kullanir; uygulama cevap vermezse Kubernetes podu yeniden baslatir.

Backend Service `LoadBalancer` tipindedir. Bu servis GKE uzerinden public IP alir ve kullanicidan gelen HTTP trafigini backend podlarina dagitir.

PostgreSQL Service `ClusterIP` tipindedir. Bu nedenle veritabani internete acilmaz; sadece cluster icinden erisilebilir.

## PV/PVC Kullanimi

`postgres-pvc.yaml`, PostgreSQL verileri icin 5Gi kalici depolama ister. `postgres-deployment.yaml` bu PVC'yi `/var/lib/postgresql/data` dizinine mount eder. Boylece PostgreSQL podu silinse bile veriler Persistent Volume uzerinde kalir.

Kontrol:

```bash
kubectl get pvc postgres-pvc
kubectl describe pvc postgres-pvc
```

## NetworkPolicy Kullanimi

`network-policy.yaml` iki temel kural tanimlar:

- PostgreSQL podlarina sadece `app=dork-backend` etiketli backend podlari TCP 5432 portundan erisebilir.
- Backend podlari 9867 portundan trafik alabilir; PostgreSQL, DNS, HTTP ve HTTPS cikis trafigine izinlidir.

Kontrol:

```bash
kubectl get networkpolicy
kubectl describe networkpolicy postgres-allow-backend-only
kubectl describe networkpolicy backend-network-policy
```

Not: GKE'de NetworkPolicy etkisinin uygulanmasi icin cluster tarafinda NetworkPolicy destegi aktif olmalidir. GKE Dataplane V2 veya NetworkPolicy etkin bir cluster bu kurallari uygular.

## Rolling Update

Rolling update, yeni imaj tag'i Deployment'a verildiginde Kubernetes'in podlari sirayla yenilemesidir. Bu projede `backend-deployment.yaml` icinde `RollingUpdate` stratejisi tanimlidir.

Manuel rolling update ornegi:

```bash
kubectl set image deployment/dork-backend \
  dork-backend-container=europe-west1-docker.pkg.dev/project-444d504d-38fb-4e0d-83e/dork-repo/projectofdork-app:NEW_TAG

kubectl rollout status deployment/dork-backend
kubectl rollout history deployment/dork-backend
```

Jenkins pipeline da deploy sirasinda ayni mantikla `kubectl set image` calistirir.

## Rollback

Son deploy sorunluysa once rollout history incelenir:

```bash
kubectl rollout history deployment/dork-backend
```

Bir onceki surume donmek icin:

```bash
kubectl rollout undo deployment/dork-backend
kubectl rollout status deployment/dork-backend
```

Belirli bir revision'a donmek icin:

```bash
kubectl rollout undo deployment/dork-backend --to-revision=2
```

## Olcekleme

Otomatik olcekleme HPA ile yapilir:

```bash
kubectl get hpa
kubectl describe hpa dork-backend-hpa
```

Bu projede backend podlari CPU kullanimina gore en az 1, en fazla 5 replika olacak sekilde olceklenir.

Manuel olcekleme demosu:

```bash
kubectl scale deployment dork-backend --replicas=3
kubectl get pods -l app=dork-backend
```

HPA aktifken uzun vadeli replika sayisini HPA tekrar kendi hedeflerine gore ayarlayabilir.

## Yerel Gelistirme

Backend'i yerelde calistirma:

```bash
cd src/backend
go mod download
go run main.go
```

Test:

```bash
cd src/backend
go test ./...
```

API ornegi:

```bash
curl "http://localhost:9867/api/dorks?domain=example.com"
```

Admin gecmis islemleri icin `/history` sayfasinda admin token girilerek admin modu acilir. Admin modu tum oturumlarin gecmisini listeler; normal mod yalnizca mevcut tarayici cookie oturumuna ait gecmisi gosterir. Komut satirindan tum gecmisi silmek icin:

```bash
curl -X POST "http://localhost:9867/" \
  -H "X-Admin-Token: uzun-rastgele-bir-token" \
  -d "action=clear_history" \
  -d "history_scope=all"
```

## Proje Yapisi

```text
.
├── Dockerfile
├── Jenkinsfile
├── README.md
├── docs
│   └── jenkins-gke.md
├── k8s
│   ├── backend-deployment.yaml
│   ├── backend-service.yaml
│   ├── admin-secret.example.yaml
│   ├── hpa.yaml
│   ├── network-policy.yaml
│   ├── postgres-deployment.yaml
│   ├── postgres-pvc.yaml
│   ├── postgres-secret.example.yaml
│   └── postgres-service.yaml
└── src
    ├── backend
    │   ├── database
    │   ├── handlers
    │   ├── models
    │   ├── go.mod
    │   └── main.go
    └── frontend
        ├── history.html
        └── index.html
```

## Yasal Uyari

Bu arac yalnizca egitim ve yetkili guvenlik testi amaciyla kullanilmalidir. Hedef sistemlerde izinsiz test yapmak hukuki ve etik sorunlara yol acabilir. Kullanim sorumlulugu tamamen kullaniciya aittir.
