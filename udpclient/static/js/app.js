const {createApp} = Vue;

// 应用配置
const APP_CONFIG = {
    POLLING_INTERVAL: 1000,
    CONNECT_TIMEOUT: 8000,
    MAX_RECENT_DEVICES: 3
};

// 工具函数
const utils = {
    async copyToClipboard(text) {
        try {
            await navigator.clipboard.writeText(text);
            return { success: true, message: '已复制!' };
        } catch (err) {
            return { success: false, message: '复制失败' };
        }
    },
    
    validatePassword(password) {
        if (!password) return '请输入新密码';
        if (password.length !== 6) return '密码必须为6位数字';
        if (!/^\d+$/.test(password)) return '密码只能包含数字';
        return null;
    },
    
    formatTime(timestamp) {
        return new Date(timestamp).toLocaleString();
    }
};

// API服务
const apiService = {
    async fetchLocalDevice() {
        const response = await fetch('/api/device');
        return response.ok ? await response.json() : null;
    },
    
    async fetchPeerStatus() {
        const response = await fetch('/api/peerStatus');
        return response.ok ? await response.json() : null;
    },
    
    async connectPeer(targetId, targetPwd) {
        const response = await fetch('/api/connect', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({ targetId, targetPwd })
        });
        return await response.json();
    },
    
    async resetPassword(newPassword) {
        const response = await fetch('/api/resetPassword', {
            method: 'POST',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify({ newPassword })
        });
        return await response.json();
    }
};

// 本地存储管理
const storageManager = {
    saveRecentDevices(devices) {
        localStorage.setItem('recentDevices', JSON.stringify(devices));
    },
    
    loadRecentDevices() {
        try {
            const saved = localStorage.getItem('recentDevices');
            return saved ? JSON.parse(saved) : [];
        } catch (e) {
            console.error('Failed to load recent devices:', e);
            return [];
        }
    },
    
    addRecentDevice(devices, newDevice) {
        const filtered = devices.filter(d => d.id !== newDevice.id);
        const updated = [newDevice, ...filtered].slice(0, APP_CONFIG.MAX_RECENT_DEVICES);
        this.saveRecentDevices(updated);
        return updated;
    },
    
    removeRecentDevice(devices, deviceId) {
        const updated = devices.filter(d => d.id !== deviceId);
        this.saveRecentDevices(updated);
        return updated;
    }
};

// Vue应用
createApp({
    data() {
        return {
            // 设备信息
            localDevice: {
                clientId: '加载中...',
                IP: '0.0.0.0',
                natType: '未知',
                password: ''
            },
            peerDevice: {
                clientId: '等待连接',
                IP: '0.0.0.0',
                alive: false,
                latency: '-'
            },
            connectionInfo: {
                mode: '断开状态',
                modeCode: 2,
                isConnected: false,
                statusText: '未连接',
                isConnecting: false,
                connectFailed: false,
                connectMessage: ''
            },
            
            // 连接相关
            targetId: '',
            connectPassword: '',
            connectionStatus: { online: false, message: '未连接' },
            isConnecting: false,
            lastConnectTime: +new Date(),
            
            // UI状态
            showConnectModal: false,
            showPasswordModal: false,
            newPassword: '',
            passwordError: false,
            passwordErrorMessage: '',
            isResetting: false,
            
            // 最近设备
            recentDevices: [],
            
            // 复制状态
            copyStatus: {
                id: '点击复制',
                pwd: '点击复制',
                peerIP: '点击复制'
            },
            
            // 轮询控制
            peerStatusInterval: null
        };
    },
    
    methods: {
        // 设备管理
        async fetchLocalDevice() {
            const device = await apiService.fetchLocalDevice();
            if (device) {
                this.localDevice = device;
            }
        },
        
        // 连接管理
        showConnectPasswordModal() {
            if (!this.targetId || this.targetId.length !== 8) {
                alert('请输入正确的连接码');
                return;
            }
            
            const savedDevice = this.recentDevices.find(d => d.id === this.targetId);
            this.connectPassword = savedDevice ? savedDevice.password : '';
            this.showConnectModal = true;
        },
        
        closeConnectPasswordModal() {
            this.showConnectModal = false;
        },
        
        async connectPeer() {
            if (!this.connectPassword || this.connectPassword.length !== 6) {
                alert('请输入6位连接密码');
                return;
            }
            
            this.isConnecting = true;
            this.closeConnectPasswordModal();
            this.connectionStatus = { online: false, message: '正在连接...' };
            this.lastConnectTime = +new Date();

            try {
                const result = await apiService.connectPeer(this.targetId, this.connectPassword);
                if (result.code !== 0) {
                    this.connectionStatus = {
                        online: false,
                        message: result.message || '连接失败'
                    };
                    this.isConnecting = false;
                }
            } catch (e) {
                console.error(e);
                alert('操作失败，请检查程序是否正在运行');
                this.connectionStatus = { online: false, message: '未连接' };
                this.isConnecting = false;
            }
        },
        
        // 状态轮询
        fetchPeerStatus() {
            if (this.peerStatusInterval) {
                clearInterval(this.peerStatusInterval);
            }
            
            const req = async () => {
                try {
                    const data = await apiService.fetchPeerStatus();
                    if (data) {
                        this.peerDevice = data.device;
                        this.connectionInfo = data.status;
                        this.updateConnectionStatus();
                    }
                } catch (e) {
                    console.error(e);
                    this.resetToDefaultState();
                }
            };
            
            this.peerStatusInterval = setInterval(req, APP_CONFIG.POLLING_INTERVAL);
            req();
        },
        
        updateConnectionStatus() {
            if (this.connectionInfo.connectFailed) {
                this.handleConnectionFailed();
            } else if (this.peerDevice.alive) {
                this.handleConnectionSuccess();
            } else if (this.connectionInfo.isConnecting || this.isConnecting) {
                // 如果后端显示正在连接，或者前端正在连接中，都显示连接状态
                this.handleConnecting();
            } else {
                this.handleDisconnected();
            }
        },
        
        handleConnectionFailed() {
            this.connectionStatus.message = this.connectionInfo.connectMessage || "连接失败";
            this.connectionStatus.online = false;
            this.isConnecting = false;
            if (this.connectionInfo.connectMessage) {
                alert(this.connectionInfo.connectMessage);
            }
        },
        
        handleConnectionSuccess() {
            if (this.targetId && this.connectPassword && this.isConnecting) {
                this.saveRecentDevice(this.targetId, this.connectPassword);
            }
            this.connectionStatus.message = this.connectionInfo.statusText;
            this.connectionStatus.online = true;
            this.isConnecting = false;
        },
        
        handleConnecting() {
            // 优先使用后端的状态消息，如果没有则使用默认消息
            const message = this.connectionInfo.connectMessage || 
                           this.connectionInfo.statusText || 
                           "正在连接...";
            this.connectionStatus.message = message;
            this.connectionStatus.online = false;
            this.isConnecting = true;
        },
        
        handleDisconnected() {
            this.connectionStatus.message = "未连接";
            this.connectionStatus.online = false;
            this.isConnecting = false;
        },
        
        resetToDefaultState() {
            this.connectionStatus = { message: "未连接", online: false };
            this.localDevice = { clientId: '加载中...', IP: '0.0.0.0', natType: '未知' };
            this.peerDevice = { clientId: '等待连接', IP: '0.0.0.0', alive: false, latency: '-' };
            this.connectionInfo = {
                mode: '断开状态',
                modeCode: 2,
                isConnected: false,
                statusText: '未连接',
                isConnecting: false,
                connectFailed: false,
                connectMessage: ''
            };
            this.isConnecting = false;
        },
        
        // 密码管理
        showResetPasswordModal() {
            this.showPasswordModal = true;
            this.newPassword = '';
            this.passwordError = false;
            this.passwordErrorMessage = '';
        },
        
        closeResetPasswordModal() {
            this.showPasswordModal = false;
        },
        
        async resetPassword() {
            if (this.isResetting) return;
            
            this.passwordError = false;
            const errorMsg = utils.validatePassword(this.newPassword);
            if (errorMsg) {
                this.passwordError = true;
                this.passwordErrorMessage = errorMsg;
                return;
            }

            this.isResetting = true;
            
            try {
                const result = await apiService.resetPassword(this.newPassword);
                if (result.code === 0) {
                    this.localDevice.password = result.password;
                    this.closeResetPasswordModal();
                } else {
                    this.passwordError = true;
                    this.passwordErrorMessage = result.message;
                }
            } catch (err) {
                console.error(err);
                this.passwordError = true;
                this.passwordErrorMessage = '修改失败，请重试';
            } finally {
                this.isResetting = false;
            }
        },
        
        // 复制功能
        async copyClientId() {
            if (this.localDevice.clientId) {
                const result = await utils.copyToClipboard(this.localDevice.clientId);
                this.updateCopyStatus('id', result.message);
            }
        },
        
        async copyPassword() {
            if (this.localDevice.password) {
                const result = await utils.copyToClipboard(this.localDevice.password);
                this.updateCopyStatus('pwd', result.message);
            }
        },
        
        async copyPeerIP() {
            if (this.peerDevice.IP) {
                const result = await utils.copyToClipboard(this.peerDevice.IP);
                this.updateCopyStatus('peerIP', result.message);
            }
        },
        
        updateCopyStatus(type, message) {
            this.copyStatus[type] = message;
            setTimeout(() => {
                this.copyStatus[type] = '点击复制';
            }, 2000);
        },
        
        // 最近设备管理
        saveRecentDevice(id, password) {
            const device = {
                id,
                password,
                lastConnected: new Date().toISOString()
            };
            this.recentDevices = storageManager.addRecentDevice(this.recentDevices, device);
        },
        
        loadRecentDevices() {
            this.recentDevices = storageManager.loadRecentDevices();
        },
        
        async quickConnect(device) {
            this.targetId = device.id;
            this.connectPassword = device.password;
            await this.connectPeer();
        },
        
        deleteDevice(deviceId) {
            if (confirm('确定要删除这个设备吗？')) {
                this.recentDevices = storageManager.removeRecentDevice(this.recentDevices, deviceId);
            }
        },
        
        // 工具方法
        formatTime(timestamp) {
            return utils.formatTime(timestamp);
        }
    },
    
    mounted() {
        this.fetchLocalDevice();
        this.fetchPeerStatus();
        this.loadRecentDevices();
    },
    
    unmounted() {
        if (this.peerStatusInterval) {
            clearInterval(this.peerStatusInterval);
        }
    }
}).mount('#app');
