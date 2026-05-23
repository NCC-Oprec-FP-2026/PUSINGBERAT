// ============================================================
// Jenkinsfile — PUSINGBERAT SIEM CI Pipeline
// ============================================================
// Triggered by: GitHub webhook on push to 'feature/day5-discord-merged', 'develop', or 'main'.
//
// Stages:
//   1. Checkout        — clone source from GitHub
//   2. Go: Lint        — run go vet (fast static check)
//   3. Go: Test        — go test + coverage.out
//   4. SonarQube Scan  — analyse Go + Vue 3 source
//   5. Quality Gate    — wait for SonarQube pass/fail
//   6. Docker Build    — build production image (backend)
//
// Jenkins credentials required (configure in Manage Credentials):
//   - sonarqube-token    → Secret Text — SonarQube user token
//   - github-credentials → Username/Password or SSH key (if private repo)
//
// Jenkins tools required (configure in Manage Jenkins → Tools):
//   - Go installation named "go-1.25"
//     (or use the docker agent below and remove the 'tools' block)
//   - SonarQube Scanner named "sonar-scanner"
//   - SonarQube Server named "sonarserver" in Manage Jenkins → System
// ============================================================

pipeline {
    agent any

    // ── Tool definitions ──────────────────────────────────────
    // These names must match what you configured in
    // Manage Jenkins → Tools.
    tools {
        go 'go-1.25'
    }

    // ── Trigger: run on feature/day5-discord-merged, develop, and main ─
    triggers {
        githubPush()
    }

    // ── Environment variables ─────────────────────────────────
    environment {
        // Docker image name for the backend.
        IMAGE_NAME    = 'pusingberat-backend'
        IMAGE_TAG     = "${env.GIT_COMMIT?.take(7) ?: 'latest'}"

        // Go module path — matches go.mod module declaration.
        GO_MODULE     = 'github.com/NCC-Oprec-FP-2026/PUSINGBERAT'

        // Path to Go coverage report (relative to workspace root).
        COVERAGE_FILE = 'backend/coverage.out'

        // SonarQube scanner binary name.
        SONAR_SCANNER = 'sonar-scanner'
    }

    // ── Pipeline options ──────────────────────────────────────
    options {
        // Abort the build if it runs longer than 20 minutes.
        timeout(time: 20, unit: 'MINUTES')
        // Keep only the last 10 builds to save disk space.
        buildDiscarder(logRotator(numToKeepStr: '10'))
        // Discard concurrent builds for the same branch.
        disableConcurrentBuilds()
        // Prepend all log lines with a timestamp.
        timestamps()
    }

    stages {

        // ── Stage 1: Checkout ─────────────────────────────────
        stage('Checkout') {
            steps {
                checkout scm
                echo "✅ Checked out branch: ${env.BRANCH_NAME} @ ${env.GIT_COMMIT?.take(7)}"
            }
        }

        // ── Stage 2: Go Lint ─────────────────────────────────
        // go vet catches suspicious constructs (printf format
        // mismatches, unreachable code, etc.) without external tools.
        stage('Go: Lint') {
            when {
                anyOf {
                    branch 'develop'
                    branch 'main'
                    branch pattern: 'feature/day5-discord-merged', comparator: 'EQUALS'
                    changeRequest()
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
        // Runs all unit tests and integration tests.
        // Produces coverage.out consumed later by SonarQube.
        //
        // NOTE: Integration tests (integration_test.go) require a
        // running PostgreSQL instance. If Jenkins does NOT have one,
        // use the build tag below to skip them:
        //   go test -tags=unit -coverprofile=coverage.out ./...
        // Remove `-tags=unit` once you wire up a test DB service.
        stage('Go: Test & Coverage') {
            when {
                anyOf {
                    branch 'develop'
                    branch 'main'
                    branch pattern: 'feature/day5-discord-merged', comparator: 'EQUALS'
                    changeRequest()
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
                    // Archive coverage report so it's visible in Jenkins UI.
                    archiveArtifacts artifacts: 'backend/coverage.out', allowEmptyArchive: true
                }
            }
        }

        // ── Stage 4: SonarQube Analysis ───────────────────────
        // Sends source + coverage data to SonarQube for analysis.
        // Requires:
        //   - A SonarQube Server named "sonarqube" configured in
        //     Manage Jenkins → System → SonarQube servers.
        //   - A credential ID "sonarqube-token" (Secret Text).
        //   - The SonarQube Scanner plugin installed.
        stage('SonarQube Analysis') {
            when {
                anyOf {
                    branch 'develop'
                    branch 'main'
                    branch pattern: 'feature/day5-discord-merged', comparator: 'EQUALS'
                    changeRequest()
                }
            }
            steps {
                // withSonarQubeEnv injects SONAR_HOST_URL and
                // SONAR_AUTH_TOKEN from the configured server.
                withSonarQubeEnv('sonarserver') {
                    sh '''
                        echo "→ Running SonarQube scanner..."
                        sonar-scanner \
                            -Dsonar.branch.name=${BRANCH_NAME} \
                            -Dsonar.go.coverage.reportPaths=backend/coverage.out

                        echo "✅ SonarQube scan submitted"
                    '''
                }
            }
        }

        // ── Stage 5: Quality Gate ────────────────────────────
        // Polls SonarQube until the analysis is complete, then
        // fails the pipeline if the Quality Gate is not passed.
        // The 5-minute timeout is a safeguard against Sonar being slow.
        stage('Quality Gate') {
            when {
                anyOf {
                    branch 'develop'
                    branch 'main'
                    branch pattern: 'feature/day5-discord-merged', comparator: 'EQUALS'
                    changeRequest()
                }
            }
            steps {
                timeout(time: 5, unit: 'MINUTES') {
                    waitForQualityGate abortPipeline: true
                }
            }
        }

        // ── Stage 6: Docker Build ─────────────────────────────
        // Delegates to docker-compose.prod.yml so the build config
        // stays in one place (Dockerfile path, context, env defaults).
        // Only runs on develop and main — not on feature branches.
        stage('Docker Build') {
            when {
                anyOf {
                    branch 'develop'
                    branch 'main'
                    branch pattern: 'feature/day5-discord-merged', comparator: 'EQUALS'
                }
            }
            steps {
                sh """
                    echo "→ Building production image via Docker Compose..."
                    docker compose \
                        -f infra/docker-compose.prod.yml \
                        build \
                        --no-cache \
                        backend

                    echo "→ Tagging image as ${IMAGE_NAME}:${IMAGE_TAG}..."
                    docker tag pusingberat-backend:latest ${IMAGE_NAME}:${IMAGE_TAG}

                    echo "✅ Docker image built: ${IMAGE_NAME}:${IMAGE_TAG}"
                """
            }
        }

    }

    // ── Post-pipeline notifications ───────────────────────────
    post {
        success {
            echo """
            ============================================
            ✅ Pipeline PASSED
            Branch : ${env.BRANCH_NAME}
            Commit : ${env.GIT_COMMIT?.take(7)}
            Build  : #${env.BUILD_NUMBER}
            ============================================
            """
        }
        failure {
            echo """
            ============================================
            ❌ Pipeline FAILED
            Branch : ${env.BRANCH_NAME}
            Commit : ${env.GIT_COMMIT?.take(7)}
            Build  : #${env.BUILD_NUMBER}
            ============================================
            """
        }
        always {
            // Clean up workspace to avoid disk bloat on the Jenkins agent.
            cleanWs()
        }
    }
}
