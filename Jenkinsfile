// ============================================================
// Jenkinsfile — PUSINGBERAT SIEM CI Pipeline
// ============================================================
// Triggered by: GitHub webhook on push to any branch.
//
// Job type: Pipeline (bukan Multibranch Pipeline).
// Karena itu BRANCH_NAME tidak tersedia secara otomatis.
// Gantinya kita gunakan GIT_BRANCH (diisi oleh Git plugin)
// yang nilainya berformat "origin/feature/integrate-be-day7-azaregon".
//
// Stages:
//   1. Checkout        — clone source dari GitHub
//   2. Go: Lint        — go vet (fast static check)
//   3. Go: Test        — go test + coverage.out
//   4. SonarQube Scan  — analisa Go + Vue 3
//   5. Quality Gate    — tunggu hasil SonarQube
//   6. Docker Build    — build image production (backend)
//
// Jenkins credentials (Manage Credentials → Global):
//   - sonarqube-token    → Secret Text — token dari SonarQube
//
// Jenkins tools (Manage Jenkins → Tools):
//   - Go:             nama "go-1.25"
//   - Sonar Scanner:  nama "sonar-scanner"
//
// Jenkins system (Manage Jenkins → System → SonarQube servers):
//   - nama server: "sonarserver"
// ============================================================

pipeline {
    agent any

    // ── Tools ────────────────────────────────────────────────
    tools {
        go 'go-1.25'
    }

    // ── Trigger ───────────────────────────────────────────────
    triggers {
        githubPush()
    }

    // ── Environment ───────────────────────────────────────────
    environment {
        IMAGE_NAME    = 'pusingberat-backend'
        IMAGE_TAG     = "${env.GIT_COMMIT?.take(7) ?: 'latest'}"
        COVERAGE_FILE = 'backend/coverage.out'

        // GIT_BRANCH diisi Git plugin: "origin/feature/integrate-be-day7-azaregon"
        // Kita normalisasi jadi "feature/integrate-be-day7-azaregon" untuk perbandingan.
        BRANCH_CLEAN  = "${env.GIT_BRANCH?.replaceFirst('origin/', '') ?: 'unknown'}"
    }

    // ── Options ───────────────────────────────────────────────
    options {
        timeout(time: 20, unit: 'MINUTES')
        buildDiscarder(logRotator(numToKeepStr: '10'))
        disableConcurrentBuilds()
        timestamps()
    }

    stages {

        // ── Stage 1: Checkout ─────────────────────────────────
        stage('Checkout') {
            steps {
                checkout scm
                // Tampilkan info branch yang sedang diproses.
                // GIT_BRANCH tersedia setelah checkout SCM.
                echo "✅ Branch: ${env.GIT_BRANCH} (clean: ${env.BRANCH_CLEAN}) @ ${env.GIT_COMMIT?.take(7)}"
            }
        }

        // ── Stage 2: Go Lint ──────────────────────────────────
        // go vet menangkap masalah umum Go tanpa external tool.
        stage('Go: Lint') {
            when {
                expression {
                    def b = env.GIT_BRANCH?.replaceFirst('origin/', '')
                    return b in ['develop', 'main', 'feature/integrate-be-day7-azaregon']
                }
            }
            steps {
                dir('backend') {
                    sh '''
                        echo "→ Running go vet..."
                        go vet ./...
                        echo "✅ go vet passed"
                    '''
                }
            }
        }

        // ── Stage 3: Go Test + Coverage ───────────────────────
        // Menghasilkan coverage.out yang dikonsumsi oleh SonarQube.
        //
        // NOTE: integration_test.go membutuhkan PostgreSQL.
        // Jika Jenkins tidak punya Postgres, tambahkan -tags=unit
        // untuk melewati integration test sementara:
        //   go test -tags=unit -coverprofile=coverage.out ./...
        stage('Go: Test & Coverage') {
            when {
                expression {
                    def b = env.GIT_BRANCH?.replaceFirst('origin/', '')
                    return b in ['develop', 'main', 'feature/integrate-be-day7-azaregon']
                }
            }
            steps {
                dir('backend') {
                    sh '''
                        echo "→ Downloading Go modules..."
                        go mod download

                        echo "→ Running tests with coverage..."
                        go test \
                            -v \
                            -timeout 120s \
                            -covermode=atomic \
                            -coverprofile=coverage.out \
                            ./...

                        echo "→ Coverage summary:"
                        go tool cover -func=coverage.out | tail -1

                        echo "✅ Tests passed"
                    '''
                }
            }
            post {
                always {
                    archiveArtifacts artifacts: 'backend/coverage.out', allowEmptyArchive: true
                }
            }
        }

        // ── Stage 4: SonarQube Analysis ───────────────────────
        // withSonarQubeEnv meng-inject SONAR_HOST_URL dan
        // SONAR_AUTH_TOKEN dari server yang dikonfigurasi di Jenkins.
        // sonar-scanner membaca sonar-project.properties dari root repo.
        stage('SonarQube Analysis') {
            when {
                expression {
                    def b = env.GIT_BRANCH?.replaceFirst('origin/', '')
                    return b in ['develop', 'main', 'feature/integrate-be-day7-azaregon']
                }
            }
            steps {
                withSonarQubeEnv('sonarserver') {
                    // Gunakan tool sonar-scanner yang dikonfigurasi di Jenkins.
                    // Ini memastikan binary ditemukan meskipun tidak ada di PATH.
                    script {
                        def scannerHome = tool 'sonarqube8.0'
                        def branchName = env.GIT_BRANCH?.replaceFirst('origin/', '') ?: 'unknown'
                        sh """
                            echo "→ Running SonarQube scanner (branch: ${branchName})..."
                            ${scannerHome}/bin/sonar-scanner \
                                -Dsonar.go.coverage.reportPaths=backend/coverage.out

                            echo "✅ SonarQube scan submitted"
                        """
                    }
                }
            }
        }

        // ── Stage 5: Quality Gate ─────────────────────────────
        // Poll SonarQube sampai analisis selesai.
        // Quality Gate hanya blocking di branch utama. Feature branch tetap
        // mengirim scan Sonar, tetapi tidak dihentikan oleh threshold coverage
        // sementara sprint masih mengejar integrasi.
        stage('Quality Gate') {
            when {
                expression {
                    def b = env.GIT_BRANCH?.replaceFirst('origin/', '')
                    return b in ['develop', 'main']
                }
            }
            steps {
                timeout(time: 5, unit: 'MINUTES') {
                    waitForQualityGate abortPipeline: true
                }
            }
        }

        // ── Stage 6: Docker Build ─────────────────────────────
        // Memanggil docker build langsung ke backend/Dockerfile.
        // TIDAK melalui docker compose karena compose akan mem-parse
        // seluruh docker-compose.prod.yml dan memvalidasi env var
        // postgres (POSTGRES_PASSWORD) yang tidak ada di Jenkins.
        // docker-compose.prod.yml tetap dipakai untuk deployment manual.
        stage('Docker Build') {
            when {
                expression {
                    def b = env.GIT_BRANCH?.replaceFirst('origin/', '')
                    return b in ['develop', 'main', 'feature/integrate-be-day7-azaregon']
                }
            }
            steps {
                sh """
                    echo "→ Building production image: ${IMAGE_NAME}:${IMAGE_TAG}..."
                    docker build \
                        -t ${IMAGE_NAME}:${IMAGE_TAG} \
                        -t ${IMAGE_NAME}:latest \
                        -f backend/Dockerfile \
                        backend/

                    echo "✅ Docker image built: ${IMAGE_NAME}:${IMAGE_TAG}"
                """
            }
        }

    }

    // ── Post ─────────────────────────────────────────────────
    post {
        success {
            echo """
            ============================================
            ✅ Pipeline PASSED
            Branch : ${env.GIT_BRANCH}
            Commit : ${env.GIT_COMMIT?.take(7)}
            Build  : #${env.BUILD_NUMBER}
            ============================================
            """
        }
        failure {
            echo """
            ============================================
            ❌ Pipeline FAILED
            Branch : ${env.GIT_BRANCH}
            Commit : ${env.GIT_COMMIT?.take(7)}
            Build  : #${env.BUILD_NUMBER}
            ============================================
            """
        }
        always {
            cleanWs()
        }
    }
}
