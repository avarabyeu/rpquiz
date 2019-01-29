import os
from flask import Flask, json, jsonify, request
from os.path import basename
from os.path import join
from padatious import IntentContainer


def create_application():
    application = Flask(__name__)
    return application


def create_container():
    container = IntentContainer('intent_cache')
    dir = os.getenv('VOCAB_DIR', '/Users/pavel_bortnik/workspace/go/src/github.com/avarabyeu/rpquiz/nlp/vocab/en-us')
    for file in os.listdir(dir):
        print(file)
        if file.endswith(".intent"):
            container.load_intent(basename(file), join(dir, file))
        elif file.endswith(".entity"):
            container.load_entity(basename(file), join(dir, file))

    container.train()
    return container


application = create_application()
container = create_container()


@application.route('/', methods=['POST'])
def index():
    json.dumps(request.json)
    content = request.json
    match = container.calc_intent(content['q'])

    return jsonify(match.__dict__)


if __name__ == '__main__':
    application.run(host='0.0.0.0', port=5000)
