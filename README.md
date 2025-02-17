# Pod Refresh Controller


## 개요

Pod Refresh Controller는 Pod의 수명을 관리하는 Controller입니다.

Pod가 오랜 기간동안 유지되면 예상하지 못한 문제가 발생할 수 있습니다. 하나의 예로 메모리 누수가 있습니다.

이 컨트롤러는 Deployment를 기준으로 Pod가 설정된 시간(PodExpirationTime)보다 오래 유지된 경우
해당 Pod를 클러스터에서 제거(Evict)하여 새로운 Pod가 실행될 수 있도록 합니다.


## 구성

- **pkg/config**
  - Controller에서 사용하는 설정을 관리합니다.
  - Informer 사용하여 설정이 저장된 ConfigMap을 관리합니다.

- **pkg/controller**
  - Pod Refresh Controller의 핵심 기능이 구현되어 있습니다.
  - 제거 대상이 되는 Pod를 조회하여 workqueue에 추가합니다.
  - Informer 사용하여 Deployment, Pod 관리합니다.

- **pkg/kubeclient**
  - Kubernetes API Client와 관련 설정을 관리합니다.

- **pkg/worker**
  - workqueue에서 대상이된 Pod를 가져와서 제거 작업을 수행합니다.

- **pkg/leaderelection**
  - HA 구성을 위해 Kubernetes의 Leader Election 기능을 구현합니다.


## 환경

아래 환경에서 테스트 되었습니다.

- Kubernetes v1.29+
- Go 1.23+


## 설치 및 빌드


### 빌드

다음 명령어를 사용하여 이미지를 빌드합니다.

```bash
docker build -t pod-refresh-controller .
```


### 배포

- Kubernetes manifest 파일은 `manifests` 디렉토리에 있습니다.
- Kustomize를 사용하여 배포하도록 구성되어 있습니다.


### 설정

ConfigMap을 사용하여 설정할 수 있습니다.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: pod-refresh-controller
data:
  podExpirationTime: "24h"
```

| 설정               | 설명            | 기본값 |
| ----------------- | --------------- | ------ |
| podExpirationTime | Pod의 최대 수명 | 24h    |


## 개발

- 로컬 환경에서 디버깅을 진행하려는 경우 `ENABLE_LOCAL_CONFIG` 환경변수를 임의의 값으로 설정합니다.
  - `~/.kube/config` 경로에 있는 설정파일을 사용합니다.


### 개발 참고 사항

- Pod Refresh Controller는 최대한 단순한 구조를 유지할 수 있도록 개발되었습니다.
  - 구현된 기능은 CRD를 필요로 하지 않기 때문에, 최소한의 Controller 구조를 사용하여 개발되었습니다.
- 배포 상황을 확인하기 위해 Deployment를 조회할 필요가 있습니다. 두 가지 방법 중 첫 번째를 선택했습니다.
  - 하향식: Deployment를 조회하고 관련이 있는 Pod를 조회하는 방법이 있습니다.
  - 상향식: Pod를 조회하고 OwnerReference를 통해 ReplicaSet -> Deployment를 조회하는 방법이 있습니다.
