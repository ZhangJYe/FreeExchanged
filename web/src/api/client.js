
import axios from 'axios';
import { appPath } from '../config/base';

const api = axios.create({
    baseURL: appPath('/v1'),
    timeout: 5000,
});

api.interceptors.request.use((config) => {
    const token = localStorage.getItem('token');
    if (token) {
        config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
});

api.interceptors.response.use(
    (response) => response.data,
    (error) => {
        if (error.response && error.response.status === 401) {
            const loginPath = appPath('/login');
            if (window.location.pathname !== loginPath) {
                localStorage.removeItem('token');
                window.location.href = loginPath;
            }
        }
        return Promise.reject(error);
    }
);

export default api;
