# GUI + Production ADS + AWS Bedrock Setup Guide

Launch the AGNTCY Directory GUI connected to the production ADS on AWS EKS, using AWS Bedrock (Claude) as the LLM via a local LiteLLM proxy.

## Prerequisites

| Tool | Install | Verify |
|------|---------|--------|
| Flutter SDK | [flutter.dev](https://flutter.dev/docs/get-started/install) | `flutter doctor` |
| Xcode (macOS) | App Store | `xcode-select -p` |
| CocoaPods | `brew install cocoapods` | `pod --version` |
| Python 3.8+ | `brew install python3` | `python3 --version` |
| LiteLLM | `pip install litellm[proxy]` | `litellm --version` |
| kubectl | `brew install kubectl` | `kubectl version --client` |
| AWS CLI v2 | [docs.aws.amazon.com](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html) | `aws --version` |
| ada | Internal Amazon tool | `ada --version` |
| Go task runner | `brew install go-task` | `task --version` |

## Step-by-Step Setup

### 1. Refresh AWS Credentials

```bash
ada credentials update --account=027020934745 --provider=isengard --role=Admin --once
```

This writes temporary credentials to `~/.aws/credentials`. Both `kubectl` and LiteLLM use these. They expire after ~1 hour.

### 2. Configure kubectl for EKS

```bash
aws eks update-kubeconfig --name ads-cluster-alpha --region us-west-2
```

**Cloud Desktop fix** (if you get `exec format error`):

```bash
sed -i '' 's|command: aws|command: /opt/homebrew/bin/aws|g' ~/.kube/config
```

Verify:

```bash
kubectl -n ads get pods
```

### 3. Start Port-Forward to ADS

```bash
kubectl -n ads port-forward svc/ads-apiserver 18888:8888 &
sleep 3
```

Verify:

```bash
/tmp/dirctl-local --server-addr localhost:18888 --auth-mode insecure search --name '*' --limit 1
```

### 4. Install and Start LiteLLM Proxy

Install (one-time):

```bash
pip install litellm[proxy]
```

Start the proxy (from the `dir/gui/` directory):

```bash
cd /Users/choppak/IdeaProjects/agentcy/src/dir/gui
litellm --config litellm_config.yaml
```

LiteLLM will start on `http://localhost:4000`. You should see output like:

```
INFO:     Uvicorn running on http://0.0.0.0:4000
```

Verify:

```bash
curl http://localhost:4000/health
```

### 5. Build the MCP Server Binary

From the `dir/` directory:

```bash
cd /Users/choppak/IdeaProjects/agentcy/src/dir
task mcp:build
```

### 6. Launch the Flutter GUI

```bash
cd /Users/choppak/IdeaProjects/agentcy/src/dir/gui

export DIRECTORY_CLIENT_SERVER_ADDRESS="localhost:18888"
export MCP_SERVER_PATH="$PWD/../bin/mcp-server"

flutter run -d macos --no-pub
```

### 7. Configure GUI Settings

Once the app launches, click the **Settings** icon and configure:

| Setting | Value |
|---------|-------|
| AI Provider | `OpenAI Compatible (Claude, etc.)` |
| API Key | `sk-1234` (any non-empty string — LiteLLM doesn't validate) |
| Base URL | `http://localhost:4000` |
| Directory Server URL | `localhost:18888` |
| Authentication Mode | `None / Insecure (Localhost)` |

Click **Save**.

### 8. Verify End-to-End

In the chat box, type:

```
list all agents
```

You should see:
- Bedrock (Claude) processes the query via LiteLLM
- The MCP server executes `agntcy_dir_search_local` against the production ADS
- Agent cards appear in the UI with results from the production directory

## Quick Restart (After Credential Expiry)

When your session expires, run these in order:

```bash
# 1. Refresh credentials
ada credentials update --account=027020934745 --provider=isengard --role=Admin --once

# 2. Update kubeconfig
aws eks update-kubeconfig --name ads-cluster-alpha --region us-west-2
sed -i '' 's|command: aws|command: /opt/homebrew/bin/aws|g' ~/.kube/config

# 3. Restart port-forward
kubectl -n ads port-forward svc/ads-apiserver 18888:8888 &

# 4. Restart LiteLLM (it picks up new credentials automatically, but restart if needed)
# If LiteLLM is still running, it will use the new credentials on next request.
# If it died, restart:
cd /Users/choppak/IdeaProjects/agentcy/src/dir/gui
litellm --config litellm_config.yaml
```

The Flutter GUI does not need to be restarted — it will reconnect automatically.

## Troubleshooting

| Problem | Symptom | Fix |
|---------|---------|-----|
| Expired AWS credentials | `kubectl` auth errors; LiteLLM returns 403 from Bedrock | Re-run `ada credentials update --account=027020934745 --provider=isengard --role=Admin --once` |
| Port-forward died | GUI shows connection error on search | Restart: `kubectl -n ads port-forward svc/ads-apiserver 18888:8888 &` |
| LiteLLM not running | GUI gets "connection refused" on chat | Start: `litellm --config litellm_config.yaml` from `dir/gui/` |
| LiteLLM config error | LiteLLM exits at startup with YAML error | Check `litellm_config.yaml` syntax; ensure `model_list` is valid |
| Bedrock model unavailable | LiteLLM returns 4xx | Verify `aws_region_name` in config; check model availability in that region |
| kubectl exec format error | `fork/exec` error with kubectl | Run: `sed -i '' 's\|command: aws\|command: /opt/homebrew/bin/aws\|g' ~/.kube/config` |
| GUI API key validation | Settings screen shows "Please enter API Key" | Enter any non-empty string (e.g., `sk-1234`) |
| MCP server not found | GUI fails to launch MCP subprocess | Run `task mcp:build` from the `dir/` directory |
| Flutter build fails | Xcode or CocoaPods errors | Run `flutter clean` then `flutter run -d macos` |
| macOS sandbox error | "Operation not permitted" when launching MCP | Ensure `macos/Runner/DebugProfile.entitlements` has `com.apple.security.app-sandbox` set to `false` |

## Architecture Overview

```
Flutter GUI ──POST /chat/completions──▶ LiteLLM (localhost:4000) ──SigV4──▶ AWS Bedrock (Claude)
     │
     └──launches──▶ MCP Server ──gRPC──▶ kubectl port-forward (localhost:18888) ──▶ ADS on EKS
```

All three local processes (port-forward, LiteLLM, GUI) must be running simultaneously.
