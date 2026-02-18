FROM python:3.11-slim

WORKDIR /app

# Install common library from local source
COPY aiwolf-nlp-common/ /tmp/common/
RUN pip install --no-cache-dir /tmp/common/ && rm -rf /tmp/common/

# Install agent
COPY aiwolf-nlp-agent-llm/ .
RUN pip install --no-cache-dir .

RUN mkdir -p ./log

CMD ["python", "src/main.py", "-c", "config/config.yml"]
