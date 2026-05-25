from pathlib import Path
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen
import os

from flask import Flask, Response, jsonify, request, send_from_directory


BASE_DIR = Path(__file__).resolve().parent
TEMPLATES_DIR = BASE_DIR / "templates"
STATIC_DIR = BASE_DIR / "static"
BACKEND_API_URL = os.getenv("BACKEND_API_URL", "http://backend:8080/api/v1").rstrip("/")

app = Flask(__name__, static_folder=str(STATIC_DIR), static_url_path="/")


@app.get("/health")
def health():
    return jsonify(
        {
            "status": "ok",
            "service": "pusingberat-flask-dev",
        }
    )


@app.route("/api/v1/<path:path>", methods=["GET", "POST", "PUT", "PATCH", "DELETE"])
def proxy_api(path):
    query = request.query_string.decode("utf-8")
    target_url = f"{BACKEND_API_URL}/{path}"
    if query:
        target_url = f"{target_url}?{query}"

    headers = {"Accept": request.headers.get("Accept", "application/json")}
    if request.content_type:
        headers["Content-Type"] = request.content_type

    proxied = Request(
        target_url,
        data=request.get_data() or None,
        headers=headers,
        method=request.method,
    )

    try:
        with urlopen(proxied, timeout=10) as backend_response:
            body = backend_response.read()
            content_type = backend_response.headers.get("Content-Type", "application/json")
            return Response(body, backend_response.status, content_type=content_type)
    except HTTPError as err:
        body = err.read()
        content_type = err.headers.get("Content-Type", "application/json")
        return Response(body, err.code, content_type=content_type)
    except URLError as err:
        return jsonify({"status": "error", "message": f"backend unavailable: {err.reason}"}), 502


@app.route("/api/v1", methods=["GET"])
def proxy_api_root():
    return proxy_api("")


def serve_template(filename):
    template = TEMPLATES_DIR / filename
    if template.exists():
        return send_from_directory(TEMPLATES_DIR, filename)
    return jsonify({"status": "not_found", "template": filename}), 404


@app.get("/")
def index():
    return serve_template("index.html")


@app.get("/alerts")
def alerts():
    return serve_template("alerts.html")


@app.get("/events")
def events():
    return serve_template("events.html")


@app.get("/logsources")
def logsource():
    return serve_template("logsources.html")


@app.get("/rules")
def rules():
    return serve_template("rules.html")


@app.get("/statics/<path:path>")
def static_files(path):
    asset = STATIC_DIR / path
    if asset.exists() and asset.is_file():
        return send_from_directory(STATIC_DIR, path)

    return serve_template("index.html")


@app.route("/<path:path>")
def spa_fallback(path):
    return serve_template("index.html")


if __name__ == "__main__":
    app.run(debug=False)
