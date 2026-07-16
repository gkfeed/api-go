FROM python:3.13-alpine

WORKDIR /app

COPY pyproject.toml ./
COPY src ./src
RUN pip install --no-cache-dir .

ENV PYTHONUNBUFFERED=1

ENTRYPOINT ["python", "-m", "main"]
