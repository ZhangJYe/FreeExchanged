
import { useState } from 'react';
import { useAuth } from '../context/AuthContext';
import { useNavigate, Link } from 'react-router-dom';

const Login = () => {
    const { login } = useAuth();
    const navigate = useNavigate();
    const [mobile, setMobile] = useState('');
    const [password, setPassword] = useState('');
    const [error, setError] = useState('');

    const handleSubmit = async (e) => {
        e.preventDefault();
        try {
            await login(mobile, password);
            navigate('/');
        } catch (e) {
            setError('Login failed: ' + (e.response?.data?.msg || e.message));
        }
    };

    return (
        <div className="card" style={{ maxWidth: 400, margin: '100px auto' }}>
            <h2>Welcome Back</h2>
            {error && <div style={{ color: 'red', marginBottom: 10 }}>{error}</div>}
            <form onSubmit={handleSubmit}>
                <input
                    type="text"
                    placeholder="Mobile"
                    value={mobile}
                    onChange={(e) => setMobile(e.target.value)}
                />
                <input
                    type="password"
                    placeholder="Password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                />
                <button type="submit" className="btn" style={{ width: '100%' }}>Login</button>
            </form>
            <div style={{ marginTop: 10, textAlign: 'center' }}>
                <Link to="/register">Create an account</Link>
            </div>
        </div>
    );
};

export default Login;
