import { useState } from 'react';
import { useAuth } from '../context/useAuth';
import { useNavigate, Link } from 'react-router-dom';

const Register = () => {
    const { register } = useAuth();
    const navigate = useNavigate();
    const [mobile, setMobile] = useState('');
    const [password, setPassword] = useState('');
    const [nickname, setNickname] = useState('');
    const [error, setError] = useState('');

    const handleSubmit = async (e) => {
        e.preventDefault();
        setError('');
        try {
            await register(mobile, password, nickname);
            navigate('/login');
        } catch (e) {
            setError(e.response?.data?.msg || 'Registration failed');
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
                    <h1 className="auth-headline">Build your rate book.</h1>
                    <div className="auth-tape" aria-hidden="true">
                        <span className="ticker-pill">WATCHLIST</span>
                        <span className="ticker-pill">LIVE QUOTES</span>
                        <span className="ticker-pill">HOT RANKS</span>
                    </div>
                </div>
            </section>

            <section className="auth-aside">
                <div className="auth-panel">
                    <h2 className="auth-title">Open account</h2>
                    <p className="auth-meta">NEW DESK PROFILE</p>
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
                                autoComplete="new-password"
                            />
                        </label>
                        <label className="form-label">
                            Nickname
                            <input
                                className="field"
                                type="text"
                                value={nickname}
                                onChange={(e) => setNickname(e.target.value)}
                                autoComplete="nickname"
                            />
                        </label>
                        <button type="submit" className="btn btn-primary">Create desk</button>
                    </form>
                    <div className="auth-footer">
                        <Link to="/login">Back to sign in</Link>
                    </div>
                </div>
            </section>
        </main>
    );
};

export default Register;
