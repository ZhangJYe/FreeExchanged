
import { createContext, useContext, useState, useEffect } from 'react';
import api from '../api/client'; // Renamed to avoid confusion with axios package
import { wsClient } from '../api/ws'; // No default export

const AuthContext = createContext();

export const useAuth = () => useContext(AuthContext);

export const AuthProvider = ({ children }) => {
    const [user, setUser] = useState(null);
    // Initialize from localStorage to prevent flash
    const [token, setToken] = useState(localStorage.getItem('token') || '');
    const [loading, setLoading] = useState(true);

    // Effect to sync Axios headers and WS connection when token changes
    useEffect(() => {
        if (token) {
            // Fetch user info
            fetchUser(token);
        } else {
            setLoading(false);
        }
    }, [token]);

    const fetchUser = async (t) => {
        try {
            const resp = await api.get('/user/info'); // Relative to baseURL /v1
            setUser(resp);
            // Connect WebSocket
            wsClient.connect(t);
        } catch (e) {
            console.error('Fetch user failed', e);
            logout();
        } finally {
            setLoading(false);
        }
    };

    const login = async (mobile, password) => {
        const resp = await api.post('/user/login', { mobile, password });
        const t = resp.token;
        localStorage.setItem('token', t);
        setToken(t);
        setUser(resp);
        // WS connection handled by effect
    };

    const logout = () => {
        localStorage.removeItem('token');
        setToken('');
        setUser(null);
        wsClient.disconnect();
    };

    const register = async (mobile, password, nickname) => {
        await api.post('/user/register', { mobile, password, nickname });
    };

    return (
        <AuthContext.Provider value={{ user, token, login, logout, register, loading }}>
            {!loading && children}
        </AuthContext.Provider>
    );
};
