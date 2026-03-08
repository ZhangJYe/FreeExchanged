
import { useState } from 'react';
import { useAuth } from '../context/AuthContext';
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
        try {
            await register(mobile, password, nickname);
            navigate('/login');
        } catch (e) {
            setError(e.response?.data?.msg || 'Registration failed');
        }
    };

    return (
        <div className="card" style={{ maxWidth: 400, margin: '100px auto' }}>
            <h2>Create Account</h2>
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
                <input
                    type="text"
                    placeholder="Nickname"
                    value={nickname}
                    onChange={(e) => setNickname(e.target.value)}
                />
                <button type="submit" className="btn" style={{ width: '100%' }}>Register</button>
            </form>
            <div style={{ marginTop: 10, textAlign: 'center' }}>
                <Link to="/login">Already have an account? Login</Link>
            </div>
        </div>
    );
};

export default Register;
