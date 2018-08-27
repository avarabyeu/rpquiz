import os
from flask import Flask, json, jsonify, request
from os.path import basename
from os.path import join
from padatious import IntentContainer

app = Flask(__name__)


@app.route('/', methods=['POST'])
def index():
    json.dumps(request.json)
    content = request.json
    match = container.calc_intent(content['q'])

    return jsonify(match.__dict__)


if __name__ == '__main__':
    container = IntentContainer('intent_cache')
    dir = os.getenv('VOCAB_DIR', '/qabot/vocab/en-us/')
    for file in os.listdir(dir):
        print(file)
        if file.endswith(".intent"):
            container.load_intent(basename(file), join(dir, file))
        elif file.endswith(".entity"):
            container.load_entity(basename(file), join(dir, file))

    container.train()

    app.run(host='0.0.0.0', port=5000)
