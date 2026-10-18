# 🚀 RAE-Suite-go (`RAE-Go v4.0`)

Natywna, bez-zależnościowa (`CGO_ENABLED=0`) implementacja ekosystemu **Silicon Oracle RAE-Suite** oraz **RAE-CRL (Cognitive Research Layer)** w języku **Go**.

---

## 🌟 Kluczowe Cechy Architektoniczne

1. **Pełna 8-Warstwowa Architektura Pamięci (`RAE-Memory` `:8000`)**:
   - **8 warstw składowania i retencji**: `sensory` (TTL decay), `working` (izolacja i szyfrowanie `AES-256-GCM` dla `INFO_CLASS=RESTRICTED`), `episodic` (100% Implicit Capture & *Szubar Mode*), `semantic` (`Knowledge Registry` & Sycon `/v2/knowledge/resolve`), `procedural` (`ProceduralChunker` & `/procedural/query`), `reflective` (4-poziomowa refleksja), `ltm` (`Long-Term Archive` & `Sealed Versions`), `graph` (`Temporal Knowledge Graph` & `Semantic Resonance`).
   - **4 Poziomy Refleksji Kognitywnej (`pkg/raecore/reflection`)**: `L1 Operational`, `L2 Structural`, `L3 Meta-Field (Szubar)`, `L4 Cognitive (The Sage)`.
   - **3 Silniki Matematyczne + `Manifold Atlas` (`pkg/raecore/math`)**: `MathL1`–`MathL3`, `MABTuner` ($\alpha, \beta, \gamma$), ramiona `classic_ib`, `hyper_37`, `fluid_100`, `neural_22`, `doc_pro` oraz wsparcie dla **Named Vectors** (`dense`, `ollama`, `openai`).
2. **Wbudowany Moduł `RAE-CRL` (`Cognitive Research Layer` `:8503` + MCP + CLI)**:
   - **7 zamkniętych typów artefaktów epistemicznych**: `epistemic_trace` (`quick_note` z **24-godzinnym okresem ochronnym `Grace Period`**), `hypothesis`, `assumption`, `experiment`, `observation`, `decision`, `reflection`.
   - **Funkcje Human-Centric**: zachowywanie wiedzy negatywnej (`FAILED_BUT_INFORMATIVE`), rozgałęzienia `Epistemic Fork` (`has_alternative_interpretation`), nadzór `PI Governance` (`approve_artifact`), detekcja konfliktów logicznych oraz pieczętowanie paczek dowodowych (`SBOM` / `VEX` / `ML-BOM`) z deterministycznym kodowaniem **`CBOR (RFC 8949)`** i `Multihash CID`.
3. **Każdy Moduł Osobno LUB Wszystkie Razem (*Modular Monolith & Microservices Dual-Mode*)**:
   - **9 samodzielnych binarek mikroserwisowych** (`cmd/rae-*`) na oryginalnych portach (`:8000`, `:8503`, `:8005`, `:8009`, `:8010`, `:8011`, `:8012`, `:8013`, `:8082`).
   - **1 zunifikowana binarka `cmd/rae`** uruchamiająca wybrane lub wszystkie moduły w pojedynczym procesie Go (`rae --all --profile=edge`).
4. **Agnostyczność Bazodanowa i Urządzeniowa (`RAE-Mesh Everywhere`)**:
   - Kompilacja ze wskaźnikiem `CGO_ENABLED=0` na `linux/amd64`, `linux/arm64`, `windows/amd64` (`.exe`) oraz `darwin/arm64` (Apple Silicon).

---

## 🛠️ Budowanie i Testowanie

```bash
# Uruchomienie testów jednostkowych i integracyjnych (z detektorem wyścigów)
make test

# Zbudowanie wszystkich 10 binarek w katalogu bin/
make build

# Kompilacja skrośna (Linux AMD64/ARM64, Windows .exe, macOS Apple Silicon)
make build-cross
```

---

## 🚀 Uruchamianie (Osobno lub Razem)

```bash
# Tryb 1: Zunifikowany (wszystkie 9 modułów w jednym procesie Go)
./bin/rae --all --profile=edge

# Tryb 2: Wybrane moduły (np. pakiet badawczy Memory + RAE-CRL + MCP)
./bin/rae --all=false --modules=memory,crl,mcp --profile=edge

# Tryb 3: Każdy moduł jako osobny proces / kontener
./bin/rae-memory   # :8000
./bin/rae-crl      # :8503 (lub CLI: ./bin/rae-crl init doktorat-kubus)
./bin/rae-mcp      # :8005 (lub Stdio: ./bin/rae-mcp --stdio)
./bin/rae-suite    # :8009
./bin/rae-quality  # :8010 (lub CLI: ./bin/rae-quality check plik.go)
./bin/rae-lab      # :8011
./bin/rae-phoenix  # :8012
./bin/rae-hive     # :8013
./bin/rae-portal   # :8082
```