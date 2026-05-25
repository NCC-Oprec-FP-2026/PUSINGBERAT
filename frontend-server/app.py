from pathlib import Path

from flask import Flask, jsonify, send_from_directory


BASE_DIR = Path(__file__).resolve().parent
DIST_DIR = BASE_DIR.parent / "frontend" / "dist"

app = Flask(__name__, static_folder=str(DIST_DIR), static_url_path="/")


@app.get("/health")
def health():
    return jsonify(
        {
            "status": "ok",
            "service": "pusingberat-flask-dev",
        }
    )


@app.route("/", defaults={"path": ""})
@app.route("/<path:path>")
def catch_all(path):
    # Serve the static file if it exists
    if path != "" and (DIST_DIR / path).exists():
        return send_from_directory(DIST_DIR, path)
    # Otherwise fallback to index.html for Vue Router SPA
    return send_from_directory(DIST_DIR, "index.html")


if __name__ == "__main__":
    app.run(debug=True)
