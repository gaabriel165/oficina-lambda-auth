# oficina-lambda-auth — Autenticação por CPF (Serverless) + API Gateway

[![CI](https://github.com/gaabriel165/oficina-lambda-auth/actions/workflows/ci.yml/badge.svg)](https://github.com/gaabriel165/oficina-lambda-auth/actions/workflows/ci.yml)
[![CD](https://github.com/gaabriel165/oficina-lambda-auth/actions/workflows/cd.yml/badge.svg)](https://github.com/gaabriel165/oficina-lambda-auth/actions/workflows/cd.yml)

Repositório da **Function Serverless de autenticação** e do **API Gateway** do Tech Challenge da pós-graduação em Arquitetura de Software (FIAP SOAT) — Fase 3. É um dos quatro repositórios do projeto:

| Repositório | Papel |
|---|---|
| [`oficina-api`](https://github.com/gaabriel165/oficina-api) | Aplicação principal (Go) executando em Kubernetes |
| [`oficina-infra-k8s`](https://github.com/gaabriel165/oficina-infra-k8s) | VPC, EKS, ECR e roles OIDC (Terraform) |
| [`oficina-infra-db`](https://github.com/gaabriel165/oficina-infra-db) | RDS PostgreSQL gerenciado (Terraform) |
| **`oficina-lambda-auth`** (este) | Lambda de autenticação por CPF, Lambda authorizer e API Gateway |

## Propósito

O cliente da oficina se identifica apenas com o **CPF**. A função `auth-cpf`:

1. Valida o CPF (dígitos verificadores, rejeita sequências repetidas);
2. Consulta a **existência** e o **status** do cliente no banco (RDS PostgreSQL, tabela `customers`);
3. Emite um **JWT** (HS256, 1 h) aceito pelas APIs protegidas.

O **API Gateway** é a única porta de entrada: expõe a rota de autenticação e faz o roteamento para a API no EKS. Rotas sensíveis (`/api/v1/*`) passam pelo **Lambda authorizer**, que valida o JWT antes de encaminhar. Rotas públicas (login administrativo, consulta de status, webhook, health e Swagger) seguem direto.

## Tecnologias

| Categoria | Tecnologias |
|---|---|
| Runtime | Go 1.25, AWS Lambda (`provided.al2023`, arm64) |
| Bibliotecas | `aws-lambda-go`, `pgx/v5`, `golang-jwt/v5`, `log/slog` (logs JSON) |
| Gateway | AWS API Gateway **HTTP API** (integração Lambda proxy + HTTP proxy para o NLB do EKS) |
| Segredos | AWS SSM Parameter Store (`/oficina-api/jwt_secret`, `/oficina-api/database_url`) |
| IaC | Terraform (state remoto em S3, lê os states de `oficina-infra-k8s` e `oficina-infra-db`) |
| CI/CD | GitHub Actions com OIDC (sem chave estática) |
| Testes | Testify |

## Como funciona

### Autenticação por CPF

```mermaid
sequenceDiagram
    autonumber
    participant C as Cliente
    participant GW as API Gateway (HTTP API)
    participant L as Lambda auth-cpf
    participant DB as RDS PostgreSQL

    C->>GW: POST /auth/cpf {"cpf": "987.654.321-00"}
    GW->>L: invoke (payload v2)
    L->>L: valida dígitos verificadores do CPF
    L->>DB: SELECT id, name, status FROM customers WHERE document = $1
    DB-->>L: cliente (status = active)
    L->>L: assina JWT HS256 (sub, cpf, name, role=customer, exp 1h)
    L-->>GW: 200 {token, token_type, expires_in, customer}
    GW-->>C: 200 {token, ...}
```

Respostas possíveis: `400` CPF inválido ou body inválido · `404` cliente não encontrado · `403` cliente inativo · `500` erro interno.

### Consumo de rota protegida

```mermaid
sequenceDiagram
    autonumber
    participant C as Cliente
    participant GW as API Gateway
    participant A as Lambda authorizer
    participant API as oficina-api (EKS)

    C->>GW: GET /api/v1/service-orders  (Authorization: Bearer <jwt>)
    GW->>A: REQUEST authorizer (identity source: header Authorization)
    A->>A: valida assinatura HS256 + exp
    A-->>GW: {isAuthorized: true, context: {sub, role}}
    GW->>API: proxy HTTP para o NLB (path preservado)
    API->>API: middleware JWT valida o mesmo token
    API-->>GW: 200 [...]
    GW-->>C: 200 [...]
```

O API Gateway cacheia a decisão do authorizer por 5 minutos para o mesmo header `Authorization`.

## Arquitetura deste repositório

```mermaid
flowchart LR
    Client[Cliente]

    subgraph AWS["AWS us-east-1"]
        subgraph GW["API Gateway HTTP API"]
            PUB["Rotas públicas<br/>GET /health · GET /swagger/*<br/>POST /api/v1/auth/login · /register<br/>GET /api/v1/service-orders/{id}/status<br/>POST /api/v1/service-orders/{id}/budget-approval"]
            AUTH["POST /auth/cpf"]
            PROT["ANY /api/v1/{proxy+}<br/>(Lambda authorizer)"]
        end
        LA["Lambda auth-cpf<br/>(na VPC)"]
        LZ["Lambda authorizer"]
        SSM[(SSM Parameter Store<br/>jwt_secret · database_url)]
        subgraph VPC["VPC (oficina-infra-k8s)"]
            NLB[NLB do Service oficina-api]
            EKS[EKS · pods oficina-api]
            RDS[(RDS PostgreSQL<br/>oficina-infra-db)]
        end
    end

    Client --> PUB
    Client --> AUTH
    Client --> PROT
    AUTH --> LA
    PROT -. valida JWT .-> LZ
    PUB --> NLB
    PROT --> NLB
    NLB --> EKS
    EKS --> RDS
    LA --> RDS
    SSM -. env vars no deploy .-> LA
    SSM -. env vars no deploy .-> LZ
```

Estrutura de pastas:

```
cmd/auth-cpf/          → handler da autenticação por CPF
cmd/authorizer/        → handler do Lambda authorizer
internal/cpf/          → validação e mascaramento de CPF
internal/customer/     → modelo + repositório PostgreSQL (pgx)
internal/token/        → emissão e verificação do JWT
internal/auth/         → caso de uso Authenticate (CPF → cliente ativo → token)
internal/httpresponse/ → respostas JSON do API Gateway
terraform/             → Lambda, IAM, API Gateway, rotas, authorizer, logs
.github/workflows/     → CI (PR) e CD (main)
```

## Execução local

```bash
make test    # go vet + testes unitários com cobertura
make build   # gera build/auth-cpf.zip e build/authorizer.zip (linux/arm64)
```

Não há Dockerfile: a função é empacotada como binário estático (`bootstrap`) em zip, formato nativo do runtime `provided.al2023`.

## Deploy

O deploy é **automático** pelo GitHub Actions:

- **`ci.yml`** (Pull Request): vet, testes, build dos pacotes, `terraform fmt/validate` e `terraform plan` contra a AWS.
- **`cd.yml`** (merge na `main`): build, `terraform apply` e smoke test em `GET /health` pelo gateway.

A branch `main` é protegida: sem commits diretos, merge somente via Pull Request com o CI verde.

### Configuração no GitHub (Settings → Secrets and variables → Actions)

| Tipo | Nome | Valor |
|---|---|---|
| Variable | `AWS_REGION` | `us-east-1` |
| Secret | `AWS_ROLE_ARN` | output `lambda_github_actions_role_arn` do repositório `oficina-infra-k8s` |

### Dependências

Este repositório é o **último** a subir e o **primeiro** a descer:

1. `oficina-infra-k8s` aplicado (state `k8s/`): VPC, subnets privadas e o parâmetro `/oficina-api/jwt_secret`.
2. `oficina-infra-db` aplicado (state `db/`): security group de cliente do banco e `/oficina-api/database_url`.
3. `oficina-api` deployado no EKS: o Service `oficina-api` cria o NLB, descoberto pela tag `kubernetes.io/service-name = oficina-api/oficina-api`. Alternativamente informe `backend_url` em `terraform.tfvars`.

### Deploy manual

```bash
make build
cd terraform
terraform init
terraform plan
terraform apply
terraform output api_gateway_url
```

## Exemplos de uso

```bash
GW=$(terraform -chdir=terraform output -raw api_gateway_url)

# 1. Autenticar com o CPF de um cliente do seed
curl -s -X POST "$GW/auth/cpf" \
  -H "Content-Type: application/json" \
  -d '{"cpf":"98765432100"}'
# {"token":"eyJ...","token_type":"Bearer","expires_in":3600,"customer":{"id":"...","name":"João Carlos Silva"}}

# 2. Consumir uma rota protegida com o token
TOKEN=$(curl -s -X POST "$GW/auth/cpf" -H "Content-Type: application/json" -d '{"cpf":"98765432100"}' | jq -r .token)
curl -s "$GW/api/v1/service-orders" -H "Authorization: Bearer $TOKEN"

# 3. Sem token → 401 do API Gateway
curl -i "$GW/api/v1/service-orders"

# 4. CPF inexistente → 404 · CPF inválido → 400 · cliente inativo → 403
curl -s -X POST "$GW/auth/cpf" -H "Content-Type: application/json" -d '{"cpf":"52998224725"}'
```

**Swagger** (com o ambiente no ar): `https://<api-id>.execute-api.us-east-1.amazonaws.com/swagger/index.html`

## Destruir o ambiente

A infraestrutura é provisionada **sob demanda** (o ambiente sobe para testes e gravação e é destruído em seguida para não gerar custo). Ordem de destroy: **este repositório primeiro**, depois `oficina-infra-db`, por último `oficina-infra-k8s`.

```bash
terraform -chdir=terraform destroy
```

## Observabilidade

- Logs estruturados em **JSON** (`log/slog`) com `request_id` do API Gateway, `event` (`auth.cpf`, `auth.authorize`), `outcome` e CPF mascarado (`987******00`) — nunca o CPF completo.
- Access logs do API Gateway em JSON (`requestId`, `routeKey`, `status`, `integrationLatency`, `responseLatency`) no CloudWatch, retenção de 7 dias.
