
import { useAuth } from '../context/AuthContext';

const Navbar = () => {
    const { user, logout } = useAuth();
    if (!user) return null;

    return (
        <nav style={{ background: '#1e293b', color: 'white', padding: '15px 20px', display: 'flex', justifyContent: 'space-between', alignItems: 'center', boxShadow: '0 2px 4px rgba(0,0,0,0.1)' }}>
            <div style={{ fontWeight: 600, fontSize: '1.2rem' }}>FreeExchanged</div>
            <div style={{ display: 'flex', alignItems: 'center' }}>
                <div style={{ marginRight: 20, display: 'flex', alignItems: 'center' }}>
                    <div style={{ width: 30, height: 30, background: '#3b82f6', borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center', marginRight: 10 }}>
                        {user.nickname ? user.nickname[0].toUpperCase() : 'U'}
                    </div>
                    <span>{user.nickname}</span>
                </div>
                <button
                    onClick={logout}
                    className="btn"
                    style={{ background: '#ef4444' }}
                >
                    Logout
                </button>
            </div>
        </nav>
    );
};
export default Navbar;
