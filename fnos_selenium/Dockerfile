FROM selenium/standalone-chrome

WORKDIR /app/fnos

RUN sudo chown -R seluser:seluser /app/fnos

COPY requirements.txt .

RUN pip install -r requirements.txt -i https://mirrors.cloud.tencent.com/pypi/simple --trusted-host mirrors.cloud.tencent.com

COPY . .

EXPOSE 5000

ENV PYTHONUNBUFFERED=1

CMD ["python", "app.py"]