import pyperclip
import pyautogui
import time
import threading
import tkinter as tk
from tkinter import ttk, messagebox
import logging
import os
import queue
import sys
import ctypes
import win32api
import win32con
import keyboard  # 全局热键

class GalgameAutoReader:
    def __init__(self):
        # 运行参数
        self.running = False
        self.last_clip = ""
        self.min_time = 1.0
        self.max_time = 5.0
        self.chars_per_second = 10
        self.click_button = "left"  # 直接使用当前鼠标位置
        self.task_queue = queue.Queue()
        self.processing = False
        self.direct_click = True

        # 日志与目录
        self.enable_logging = False
        self.log_file_handler = None
        self.log_dir = "galgame_reader_logs"

        # 热键
        self.start_hotkey = 'f8'
        self.stop_hotkey = 'f9'
        self.start_hotkey_handle = None
        self.stop_hotkey_handle = None

        # 初始化日志
        self.setup_logging()

        # 创建 UI
        self.root = tk.Tk()
        self.root.title("Galgame自动阅读器")
        try:
            self.root.iconbitmap("auto.ico")
        except Exception:
            pass
        self.create_widgets()
        self.root.update_idletasks()
        req_w, req_h = self.root.winfo_reqwidth(), self.root.winfo_reqheight()
        self.root.geometry(f"{req_w}x{req_h}")
        self.root.resizable(False, False)
        self.root.protocol("WM_DELETE_WINDOW", self.quit_program)

        # 后台线程
        self.clipboard_thread = threading.Thread(target=self.monitor_clipboard, daemon=True)
        self.clipboard_thread.start()
        self.task_thread = threading.Thread(target=self.process_tasks, daemon=True)
        self.task_thread.start()

        # 主循环
        self.root.mainloop()
    
    def setup_logging(self):
        # 基础控制台日志（不保存文件，除非用户勾选）
        if not logging.getLogger().handlers:
            logging.basicConfig(level=logging.INFO,
                                format='%(asctime)s - %(levelname)s - %(message)s')
        os.makedirs(self.log_dir, exist_ok=True)
        logging.info("程序启动 (未写入文件，等待用户开启日志保存)")

    def enable_file_logging(self):
        if self.log_file_handler is None:
            log_file = os.path.join(self.log_dir, f"galgame_reader_{time.strftime('%Y%m%d_%H%M%S')}.log")
            try:
                fh = logging.FileHandler(log_file, mode='a', encoding='utf-8')
                fmt = logging.Formatter('%(asctime)s - %(levelname)s - %(message)s')
                fh.setFormatter(fmt)
                logging.getLogger().addHandler(fh)
                self.log_file_handler = fh
                logging.info(f"已启用文件日志: {log_file}")
            except Exception as e:
                messagebox.showwarning("日志", f"启用日志失败: {e}")

    def disable_file_logging(self):
        if self.log_file_handler is not None:
            try:
                logging.getLogger().removeHandler(self.log_file_handler)
                self.log_file_handler.close()
            except Exception:
                pass
            self.log_file_handler = None
            logging.info("已停止文件日志保存")
    
    def create_widgets(self):
      # 状态
      self.status_var = tk.StringVar(value="状态: 未运行")
      ttk.Label(self.root, textvariable=self.status_var, font=("Arial", 10)).pack(pady=10)

      # 控制按钮
      btn_frame = ttk.Frame(self.root)
      btn_frame.pack(pady=8)
      self.start_btn = ttk.Button(btn_frame, text="开始", command=self.start)
      self.start_btn.pack(side=tk.LEFT, padx=5)
      self.stop_btn = ttk.Button(btn_frame, text="停止", command=self.stop, state=tk.DISABLED)
      self.stop_btn.pack(side=tk.LEFT, padx=5)

      # 设置分组
      settings = ttk.LabelFrame(self.root, text="设置")
      settings.pack(pady=6, padx=10, fill=tk.X)

      # 速度
      speed_row = ttk.Frame(settings)
      speed_row.pack(fill=tk.X, pady=4)
      ttk.Label(speed_row, text="阅读速度 (字/秒):").pack(side=tk.LEFT)
      self.speed_var = tk.IntVar(value=self.chars_per_second)
      ttk.Entry(speed_row, textvariable=self.speed_var, width=6).pack(side=tk.LEFT, padx=5)

      # 时间
      time_row = ttk.Frame(settings)
      time_row.pack(fill=tk.X, pady=4)
      ttk.Label(time_row, text="最短 (秒):").pack(side=tk.LEFT)
      self.min_time_var = tk.DoubleVar(value=self.min_time)
      ttk.Entry(time_row, textvariable=self.min_time_var, width=6).pack(side=tk.LEFT, padx=5)
      ttk.Label(time_row, text="最长 (秒):").pack(side=tk.LEFT, padx=(10, 0))
      self.max_time_var = tk.DoubleVar(value=self.max_time)
      ttk.Entry(time_row, textvariable=self.max_time_var, width=6).pack(side=tk.LEFT, padx=5)


      # 鼠标按钮
      mouse_row = ttk.Frame(settings)
      mouse_row.pack(fill=tk.X, pady=4)
      ttk.Label(mouse_row, text="鼠标按钮:").pack(side=tk.LEFT)
      self.btn_var = tk.StringVar(value=self.click_button)
      ttk.Radiobutton(mouse_row, text="左键", value="left", variable=self.btn_var).pack(side=tk.LEFT, padx=5)
      ttk.Radiobutton(mouse_row, text="右键", value="right", variable=self.btn_var).pack(side=tk.LEFT, padx=5)

      # 高级
      advanced_row = ttk.Frame(settings)
      advanced_row.pack(fill=tk.X, pady=4)
      self.direct_click_var = tk.BooleanVar(value=self.direct_click)
      ttk.Checkbutton(advanced_row, text="使用直接点击", variable=self.direct_click_var).pack(side=tk.LEFT)
      self.log_var = tk.BooleanVar(value=self.enable_logging)
      ttk.Checkbutton(advanced_row, text="保存日志", variable=self.log_var).pack(side=tk.LEFT, padx=12)

      # 热键
      hotkey_frame = ttk.LabelFrame(self.root, text="热键")
      hotkey_frame.pack(pady=6, padx=10, fill=tk.X)
      ttk.Label(hotkey_frame, text="开始热键:").grid(row=0, column=0, padx=5, pady=3, sticky=tk.W)
      self.start_hotkey_var = tk.StringVar(value=self.start_hotkey)
      ttk.Entry(hotkey_frame, textvariable=self.start_hotkey_var, width=18).grid(row=0, column=1, padx=5, pady=3, sticky=tk.W)
      ttk.Label(hotkey_frame, text="停止热键:").grid(row=1, column=0, padx=5, pady=3, sticky=tk.W)
      self.stop_hotkey_var = tk.StringVar(value=self.stop_hotkey)
      ttk.Entry(hotkey_frame, textvariable=self.stop_hotkey_var, width=18).grid(row=1, column=1, padx=5, pady=3, sticky=tk.W)
      ttk.Button(hotkey_frame, text="应用", command=self.apply_hotkeys).grid(row=0, column=2, rowspan=2, padx=10)

      # 提示
      tips = ttk.Label(self.root, foreground="blue", justify="center",
                 text=("提示: 使用热键开始/停止自动阅读\n"
                     "复制游戏文本后自动计算等待并点击\n"
                     "若点击无效尝试启用'使用直接点击'\n"
                     f"停留时间限制: 最短{self.min_time}秒, 最长{self.max_time}秒"),
                 font=("Arial", 9))
      tips.pack(pady=8)

      self.register_hotkeys()
    # 已移除托盘相关方法

    def register_hotkeys(self):
        try:
            if self.start_hotkey_handle is not None:
                keyboard.remove_hotkey(self.start_hotkey_handle)
            if self.stop_hotkey_handle is not None:
                keyboard.remove_hotkey(self.stop_hotkey_handle)
            self.start_hotkey_handle = keyboard.add_hotkey(self.start_hotkey, self.start)
            self.stop_hotkey_handle = keyboard.add_hotkey(self.stop_hotkey, self.stop)
            logging.info(f"注册热键: 开始[{self.start_hotkey}] 停止[{self.stop_hotkey}]")
        except Exception as e:
            logging.error(f"注册热键失败: {e}")
            try:
                messagebox.showerror("热键错误", f"注册热键失败: {e}")
            except Exception:
                pass

    def apply_hotkeys(self):
        start = self.start_hotkey_var.get().strip().lower()
        stop = self.stop_hotkey_var.get().strip().lower()
        if not start or not stop:
            messagebox.showwarning("提示", "热键不能为空")
            return
        if start == stop:
            messagebox.showwarning("提示", "开始与停止热键不能相同")
            return
        self.start_hotkey = start
        self.stop_hotkey = stop
        self.register_hotkeys()
        messagebox.showinfo("热键", f"已应用:\n开始: {self.start_hotkey}\n停止: {self.stop_hotkey}")
    
    
    def start(self, icon=None, item=None):
        if self.running:
            return
        self.running = True
        self.status_var.set("状态: 运行中")
        self.start_btn.config(state=tk.DISABLED)
        self.stop_btn.config(state=tk.NORMAL)
        # 读取最新配置
        try:
            self.chars_per_second = max(1, int(self.speed_var.get()))
            self.min_time = max(0.1, float(self.min_time_var.get()))
            self.max_time = max(self.min_time, float(self.max_time_var.get()))
        except Exception:
            messagebox.showwarning("提示", "参数格式错误，已使用默认值")
        self.click_button = self.btn_var.get()
        self.direct_click = self.direct_click_var.get()
        # 日志选项
        want_log = self.log_var.get()
        if want_log and not self.enable_logging:
            self.enable_file_logging()
        if (not want_log) and self.enable_logging:
            self.disable_file_logging()
        self.enable_logging = want_log
        logging.info(f"开始: 速度{self.chars_per_second}字/秒, 时间{self.min_time}-{self.max_time}")
        messagebox.showinfo("提示", "自动阅读已开始，复制文本触发计时点击。")
    
    def stop(self, icon=None, item=None):
        if not self.running:
            return
        self.running = False
        self.status_var.set("状态: 已停止")
        self.start_btn.config(state=tk.NORMAL)
        self.stop_btn.config(state=tk.DISABLED)
        logging.info("自动阅读已停止")
    
    def quit_program(self, icon=None, item=None):
        self.running = False
        try:
            keyboard.unhook_all_hotkeys()
        except Exception:
            pass
        # 关闭文件日志句柄
        if self.log_file_handler is not None:
            try:
                logging.getLogger().removeHandler(self.log_file_handler)
                self.log_file_handler.close()
            except Exception:
                pass
        try:
            self.root.destroy()
        except Exception:
            pass
        logging.info("程序已退出")
        os._exit(0)
    
    def monitor_clipboard(self):
        while True:
            try:
                if not self.running:
                    time.sleep(0.5)
                    continue
                current_clip = pyperclip.paste().strip()
                if current_clip and current_clip != self.last_clip:
                    self.last_clip = current_clip
                    length = len(current_clip)
                    calculated = length / max(1, self.chars_per_second)
                    wait_time = max(self.min_time, min(calculated, self.max_time))
                    self.task_queue.put({"length": length, "wait_time": wait_time})
                    self.root.after(0, lambda l=length, w=wait_time: self.status_var.set(
                        f"状态: 运行中 - {l}字, 等待{w:.1f}秒"))
                    logging.info(f"检测到文本 {length} 字, 等待 {wait_time:.2f}s")
                time.sleep(0.2)
            except Exception as e:
                logging.error(f"监控剪贴板出错: {e}")
                time.sleep(1)
    
    def process_tasks(self):
        while True:
            try:
                if not self.running or self.processing:
                    time.sleep(0.05)
                    continue
                if self.task_queue.empty():
                    time.sleep(0.05)
                    continue
                self.processing = True
                task = self.task_queue.get()
                # 分段等待，允许在等待过程中按停止键立即终止
                total_wait = task["wait_time"]
                waited = 0.0
                slice_len = 0.1
                while waited < total_wait and self.running:
                    step = min(slice_len, total_wait - waited)
                    time.sleep(step)
                    waited += step
                # 如果仍在运行再执行点击，并只在仍运行时恢复状态文字
                if self.running:
                    self.perform_click()
                    self.root.after(0, lambda: self.running and self.status_var.set("状态: 运行中"))
                self.processing = False
            except Exception as e:
                logging.error(f"处理任务出错: {e}")
                self.processing = False
                time.sleep(0.5)
    
    def perform_click(self):
        """在当前鼠标位置执行点击。"""
        try:
            x, y = pyautogui.position()
            if self.direct_click:
                self.direct_click_method(x, y)
            else:
                pyautogui.click(x=x, y=y, button=self.click_button)
            logging.info(f"点击 {self.click_button} @ ({x},{y})")
        except Exception as e:
            logging.error(f"点击出错: {e}")
    
    def direct_click_method(self, x, y):
        ctypes.windll.user32.SetCursorPos(x, y)
        if self.click_button == "left":
            down, up = win32con.MOUSEEVENTF_LEFTDOWN, win32con.MOUSEEVENTF_LEFTUP
        else:
            down, up = win32con.MOUSEEVENTF_RIGHTDOWN, win32con.MOUSEEVENTF_RIGHTUP
        win32api.mouse_event(down, x, y, 0, 0)
        time.sleep(0.03)
        win32api.mouse_event(up, x, y, 0, 0)

if __name__ == "__main__":  # 入口判断，确保脚本直接运行时执行
    try:  # 捕获启动异常
        # 检查是否以管理员权限运行
        if ctypes.windll.shell32.IsUserAnAdmin() == 0:  # 调用系统API判断管理员权限
            # 如果不是管理员，尝试以管理员权限重新启动
            ctypes.windll.shell32.ShellExecuteW(None, "runas", sys.executable, " ".join(sys.argv), None, 1)  # 提升权限重新启动
            sys.exit(0)  # 退出当前普通权限进程
        
        app = GalgameAutoReader()  # 创建应用实例并启动
    except Exception as e:  # 捕获任意异常
        print(f"程序启动失败: {e}")  # 控制台输出启动失败信息