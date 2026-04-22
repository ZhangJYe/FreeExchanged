import { useAuth } from '../context/useAuth';

const ExitIcon = () => (
    <svg className="icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
        <path d="M9 7V5.8C9 4.8 9.8 4 10.8 4H18v16h-7.2A1.8 1.8 0 0 1 9 18.2V17" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
        <path d="M13 12H3m0 0 3.5-3.5M3 12l3.5 3.5" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
);

const Navbar = () => {
    const { user, logout } = useAuth();
    if (!user) return null;

    const initial = user.nickname ? user.nickname[0].toUpperCase() : 'U';

    return (
        <nav className="topbar">
            <div className="brand-lockup">
                <div className="brand-mark">FX</div>
                <div>
                    <h1 className="brand-name">FreeExchanged</h1>
                    <p className="brand-meta">LIVE FX WORKSTATION</p>
                </div>
            </div>

            <div className="nav-actions">
                <div className="user-chip">
                    <div className="avatar">{initial}</div>
                    <span className="user-name">{user.nickname || 'Trader'}</span>
                </div>
                <button type="button" className="btn btn-quiet" onClick={logout}>
                    <ExitIcon />
                    Logout
                </button>
            </div>
        </nav>
    );
};

export default Navbar;
