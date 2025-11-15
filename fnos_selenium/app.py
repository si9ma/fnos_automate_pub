from flask import Flask, request, jsonify
import threading
import json
import os
from selenium_service import SeleniumLoginService

app = Flask(__name__)
app.json.ensure_ascii = False
selenium_service = None
service_lock = threading.Lock()

# print environment variables for debugging
print(f"环境变量: FNOS_LOGIN_URL={os.getenv('FNOS_LOGIN_URL')}, FNOS_USERNAME={os.getenv('FNOS_USERNAME')}, FNOS_PASSWORD={os.getenv('FNOS_PASSWORD')}")

def init_selenium_service():
    """初始化Selenium服务"""
    global selenium_service
    with service_lock:
        if selenium_service is None:
            selenium_service = SeleniumLoginService()
    return selenium_service

@app.route('/health', methods=['GET'])
def health_check():
    """健康检查端点"""
    service = init_selenium_service()
    page_info = service.get_page_info() if service.driver else None
    return jsonify({
        "status": "service_running",
        "browser_ready": service.driver is not None,
        "logged_in": service.is_logged_in if service.driver else False,
        "page_info": page_info
    })

@app.route('/login', methods=['GET'])
def login():
    """登录端点"""
    try:
        service = init_selenium_service()

        login_url = os.getenv("FNOS_LOGIN_URL", '')
        username = os.getenv('FNOS_USERNAME', '')
        password = os.getenv('FNOS_PASSWORD', '')
        
        # 执行登录
        result = service.login(
            login_url=login_url,
            username=username,
            password=password,
        )
        
        return jsonify(result)
        
    except Exception as e:
        return jsonify({"status": "error", "message": str(e)}), 500

@app.route('/gen_photo_sign', methods=['POST'])
def execute_gen_sign_script():
    """执行脚本端点"""
    try:
        data = request.get_json()
        
        data_str = json.dumps(data)
        
        service = init_selenium_service()
        
        result = service.execute_gen_sign_script(params=data_str)
        
        return jsonify(result)
        
    except Exception as e:
        return jsonify({"status": "error", "message": str(e)}), 500

@app.route('/info', methods=['GET'])
def get_info():
    """获取当前浏览器信息"""
    service = init_selenium_service()
    info = service.get_page_info()
    return jsonify(info if info else {"status": "browser_not_initialized"})

@app.route('/shutdown', methods=['POST'])
def shutdown():
    """关闭浏览器端点"""
    try:
        service = init_selenium_service()
        service.close()
        return jsonify({"status": "success", "message": "浏览器已关闭"})
    except Exception as e:
        return jsonify({"status": "error", "message": str(e)}), 500

if __name__ == '__main__':
    # 初始化服务
    init_selenium_service()
    
    # 启动Flask应用
    app.run(host='0.0.0.0', port=5000, debug=False, threaded=True)