from pathlib import Path

from flask import Flask, jsonify, send_from_directory
import flask


TEMPLATES_DIR = Path("templates")
STATIC_DIR = Path("static")

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
    index_file = TEMPLATES_DIR / "index.html"
    if index_file.exists():
        # return flask.re
        return send_from_directory(TEMPLATES_DIR, "index.html")

@app.get("/alerts")
def alerts():
    index_file = TEMPLATES_DIR / "alerts.html"
    if index_file.exists():
        # return flask.re
        return send_from_directory(TEMPLATES_DIR, "alerts.html")

@app.get("/events")
def events():
    index_file = TEMPLATES_DIR / "events.html"
    if index_file.exists():
        # return flask.re
        return send_from_directory(TEMPLATES_DIR, "events.html")

@app.get("/logsources")
def logsource():
    index_file = TEMPLATES_DIR / "logsources.html"
    if index_file.exists():
        # return flask.re
        return send_from_directory(TEMPLATES_DIR, "logsources.html")

@app.get("/rules")
def rules():
    index_file = TEMPLATES_DIR / "rules.html"
    if index_file.exists():
        # return flask.re
        return send_from_directory(TEMPLATES_DIR, "rules.html")



@app.get("/statics/<path:path>")
def static_files(path):
    asset = STATIC_DIR / path
    if asset.exists() and asset.is_file():
        print(asset)
        return send_from_directory(STATIC_DIR, path)

    index_file = TEMPLATES_DIR / "index.html"
    if index_file.exists():
        return send_from_directory(TEMPLATES_DIR, "index.html")

    return jsonify({"status": "not_found", "path": path}), 404



if __name__ == '__main__':
    app.run(debug=True)
