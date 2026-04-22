import { useCallback, useEffect, useState } from 'react';
import api from '../api/client';
import { wsClient } from '../api/ws';
import { AuthContext } from './auth-context';

export const AuthProvider = ({ children }) => {
    const [user, setUser] = useState(null);
    const [token, setToken] = useState(localStorage.getItem('token') || '');
    const [loading, setLoading] = useState(true);

    const logout = useCallback(() => {
        localStorage.removeItem('token');
        setToken('');
        setUser(null);
        wsClient.disconnect();
    }, []);

    const fetchUser = useCallback(async (activeToken) => {
        try {
            const resp = await api.get('/user/info');
            setUser(resp);
            wsClient.connect(activeToken);
        } catch (e) {
            console.error('Fetch user failed', e);
            logout();
        } finally {
            setLoading(false);
        }
    }, [logout]);

    useEffect(() => {
        if (token) {
            fetchUser(token);
        } else {
            setLoading(false);
        }
    }, [fetchUser, token]);

    const login = async (mobile, password) => {
        const resp = await api.post('/user/login', { mobile, password });
        const nextToken = resp.token;
        localStorage.setItem('token', nextToken);
        setToken(nextToken);
        setUser(resp);
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
