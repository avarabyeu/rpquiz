## This one is based on Debian
FROM golang:1.11-alpine

RUN apk add --update --no-cache \
    supervisor curl cmake fann-dev wget unzip python3-dev python3 swig \
    alpine-sdk \
    ca-certificates \
    tzdata

RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh && \
    go get -v github.com/oxequa/realize github.com/alecthomas/gometalinter && \
    gometalinter --install

RUN mkdir /fann && \
    wget -O /fann/fann.zip http://sourceforge.net/projects/fann/files/fann/2.2.0/FANN-2.2.0-Source.zip/download && \
    unzip /fann/fann.zip -d /fann && \
    cd /fann/FANN-2.2.0-Source && cmake . && make install
RUN pip3 install --upgrade pip && pip3 install flask padatious

ENV WORKDIR=/go/src/github.com/avarabyeu/rpquiz/
WORKDIR $WORKDIR


#COPY glide.lock glide.yaml Makefile ./
COPY Gopkg.toml Gopkg.lock Makefile ./
RUN dep ensure --vendor-only

#RUN make build

ENV VOCAB_DIR=${WORKDIR}/nlp/vocab/en-us



## Building python stuff
ADD supervisor-dev.ini /etc/supervisor-dev.ini

ENTRYPOINT ["/usr/bin/supervisord", "--nodaemon", "--configuration", "/etc/supervisor-dev.ini"]
