from pathlib import Path

from flask import Flask, jsonify, send_from_directory

DIST_DIR = Path("/app/frontend/dist")

app = Flask(__name__, static_folder=None)


@app.get("/health")
def health():
    return jsonify(
        {
            "status": "ok",
            "service": "pusingberat-flask-dev",
        }
    )


@app.get("/")
def index():
    index_file = DIST_DIR / "index.html"
    if index_file.exists():
        return send_from_directory(DIST_DIR, "index.html")

    return jsonify(
        {
            "status": "ok",
            "service": "pusingberat-flask-dev",
            "message": "Vue dev server is available on http://localhost:5173",
        }
    )


@app.get("/<path:path>")
def static_files(path):
    asset = DIST_DIR / path
    if asset.exists() and asset.is_file():
        return send_from_directory(DIST_DIR, path)

    index_file = DIST_DIR / "index.html"
    if index_file.exists():
        return send_from_directory(DIST_DIR, "index.html")

    return jsonify({"status": "not_found", "path": path}), 404
