from pathlib import Path

from flask import Flask, jsonify, send_from_directory


BASE_DIR = Path(__file__).resolve().parent
TEMPLATES_DIR = BASE_DIR / "templates"
STATIC_DIR = BASE_DIR / "static"

app = Flask(__name__, static_folder=None)


@app.get("/health")
def health():
    return jsonify(
        {
            "status": "ok",
            "service": "pusingberat-flask-dev",
        }
    )


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

    return jsonify({"status": "not_found", "path": path}), 404


if __name__ == "__main__":
    app.run(debug=True)
