from selenium import webdriver
from selenium.webdriver.common.by import By
from selenium.webdriver.support.ui import WebDriverWait
from selenium.webdriver.support import expected_conditions as EC
from selenium.webdriver.chrome.options import Options
from selenium.webdriver.chrome.service import Service
import time
import threading

class SeleniumLoginService:
    def __init__(self, driver_path=None):
        self.lock = threading.Lock()  # 线程安全锁
        self.driver_path = driver_path
        self.sign_driver = self.setup_driver(init_url="file:///app/fnos/fnos_fake.html")
        self.login_driver = self.setup_driver()
    
    def setup_driver(self, init_url=None):
        """配置浏览器驱动"""
        chrome_options = webdriver.ChromeOptions()
        chrome_options.add_argument('--no-sandbox')
        chrome_options.add_argument('--disable-dev-shm-usage')
        chrome_options.add_argument('--headless')
        
        driver = None
        try:
            if self.driver_path:
                service = Service(executable_path=self.driver_path)
                driver = webdriver.Chrome(service=service, options=chrome_options)
            else:
                driver = webdriver.Chrome(options=chrome_options)
            
            if init_url:
                driver.get(init_url)
            return driver
            
        except Exception as e:
            raise Exception(f"浏览器驱动初始化失败: {str(e)}")
    
    def login(self, login_url, username, password):
        with self.lock:
            try:
                self.login_driver.get(login_url)

                time.sleep(5)

                if "login" not in self.login_driver.current_url.lower():
                    cookie = self.login_driver.get_cookies()
                    return {"status": "success", "message": "已登录，无需重复登录", "current_url": self.login_driver.current_url, "data": cookie}
                
                username_field = self.login_driver.find_element(By.ID, "username")
                username_field.clear()
                username_field.send_keys(username)
                
                password_field = self.login_driver.find_element(By.ID, "password")
                password_field.clear()
                password_field.send_keys(password)
                
                # click keep me logged in checkbox
                checkbox = self.login_driver.find_element(By.XPATH, "//input[@type='checkbox']")
                actions = webdriver.ActionChains(self.login_driver)
                actions.move_to_element(checkbox).click().perform()
                
                submit_button = self.login_driver.find_element(By.XPATH, "//button[@type='submit']")
                submit_button.click()

                time.sleep(2)
                
                current_url = self.login_driver.current_url
                if "login" not in current_url and "error" not in current_url.lower():
                    print("登录成功！")
                    cookie = self.login_driver.get_cookies()
                    return {"status": "success", "message": "登录成功", "current_url": current_url, "data": cookie}
                else:
                    return {"status": "error", "message": f"登录可能失败，请检查凭据 {current_url}"}
                    
            except Exception as e:
                return {"status": "error", "message": f"登录过程出错: {str(e)}"}
    
    def execute_gen_sign_script(self, params):
        try:
            result = self.sign_driver.execute_script(f"return YZ({params});")
            return {"status": "success", "result": result, "current_url": self.sign_driver.current_url}

        except Exception as e:
            return {"status": "error", "message": f"脚本执行失败: {str(e)}"}
    
    def get_page_info(self):
        """获取当前页面信息"""
        res = {}
        if self.sign_driver:
            res.update({
                "sign_driver_url": self.sign_driver.current_url,
                "sign_driver_title": self.sign_driver.title
            })
        if self.login_driver:
            res.update({
                "login_driver_url": self.login_driver.current_url,
                "login_driver_title": self.login_driver.title
            })

        return res
    
    def close(self):
        """关闭浏览器"""
        if self.sign_driver:
            self.sign_driver.quit()
        if self.login_driver:
            self.login_driver.quit()