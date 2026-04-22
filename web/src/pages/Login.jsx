import { useState } from 'react';
import { useAuth } from '../context/useAuth';
import { useNavigate, Link } from 'react-router-dom';

const Login = () => {
    const { login } = useAuth();
    const navigate = useNavigate();
    const [mobile, setMobile] = useState('');
    const [password, setPassword] = useState('');
    const [error, setError] = useState('');

    const handleSubmit = async (e) => {
        e.preventDefault();
        setError('');
        try {
            await login(mobile, password);
            navigate('/');
        } catch (e) {
            setError('Login failed: ' + (e.response?.data?.msg || e.message));
        }
    };

    return (
        <main className="auth-page">
            <section className="auth-market">
                <div className="auth-brand">
                    <div className="brand-mark">FX</div>
                    <span>FreeExchanged</span>
                </div>
                <div>
                    <h1 className="auth-headline">Live rates, clean signals.</h1>
                    <div className="auth-tape" aria-hidden="true">
                        <span className="ticker-pill">USD/CNY</span>
                        <span className="ticker-pill">EUR/USD</span>
                        <span className="ticker-pill">GBP/USD</span>
                        <span className="ticker-pill">JPY/USD</span>
                    </div>
                </div>
            </section>

            <section className="auth-aside">
                <div className="auth-panel">
                    <h2 className="auth-title">Sign in</h2>
                    <p className="auth-meta">SECURE MARKET ACCESS</p>
                    {error && <div className="notice notice-error">{error}</div>}
                    <form className="auth-form" onSubmit={handleSubmit}>
                        <label className="form-label">
                            Mobile
                            <input
                                className="field"
                                type="text"
                                value={mobile}
                                onChange={(e) => setMobile(e.target.value)}
                                autoComplete="tel"
                            />
                        </label>
                        <label className="form-label">
                            Password
                            <input
                                className="field"
                                type="password"
                                value={password}
                                onChange={(e) => setPassword(e.target.value)}
                                autoComplete="current-password"
                            />
                        </label>
                        <button type="submit" className="btn btn-primary">Enter desk</button>
                    </form>
                    <div className="auth-footer">
                        <Link to="/register">Create account</Link>
                    </div>
                </div>
            </section>
        </main>
    );
};

export default Login;
