
import { useEffect, useState } from 'react';
import api from '../api/client';

const Watchlist = () => {
    const [items, setItems] = useState([]);
    const [newPair, setNewPair] = useState('USD/CNY');
    const [message, setMessage] = useState('');
    const [loading, setLoading] = useState(false);

    const fetchList = async () => {
        try {
            const res = await api.get('/watchlist/list');
            setItems(res.items || []);
        } catch (e) {
            console.error(e);
        }
    };

    useEffect(() => {
        fetchList();
        const timer = setInterval(fetchList, 5000); // Auto-refresh rates every 5s
        return () => clearInterval(timer);
    }, []);

    const handleAdd = async (e) => {
        e.preventDefault();
        if (!newPair) return;
        setLoading(true);
        try {
            await api.post('/watchlist/add', { currency_pair: newPair });
            setMessage('Added successfully');
            setNewPair('');
            fetchList();
        } catch (e) {
            setMessage('Error: ' + (e.response?.data?.msg || e.message));
        } finally {
            setLoading(false);
            setTimeout(() => setMessage(''), 3000);
        }
    };

    const handleRemove = async (pair) => {
        try {
            await api.post('/watchlist/remove', { currency_pair: pair });
            fetchList();
        } catch (e) {
            console.error(e);
        }
    };

    return (
        <div className="card">
            <h3 style={{ marginTop: 0 }}>My Watchlist</h3>
            {message && <div style={{ marginBottom: 10, color: message.includes('Error') ? 'red' : 'green', fontSize: '0.9em' }}>{message}</div>}

            <form onSubmit={handleAdd} style={{ display: 'flex', gap: 10, marginBottom: 20 }}>
                <input
                    value={newPair}
                    onChange={e => setNewPair(e.target.value.toUpperCase())}
                    placeholder="Pair (e.g. USD/CNY)"
                    style={{ flex: 1, textTransform: 'uppercase' }}
                    disabled={loading}
                />
                <button type="submit" className="btn" disabled={loading}>
                    {loading ? 'Adding...' : 'Add'}
                </button>
            </form>

            {items.length === 0 ? (
                <p style={{ color: '#666', textAlign: 'center', padding: 20 }}>No items in watchlist.</p>
            ) : (
                <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                    <thead>
                        <tr style={{ borderBottom: '2px solid #f1f5f9', color: '#64748b', fontSize: '0.9em' }}>
                            <th style={{ textAlign: 'left', padding: '10px 8px' }}>Pair</th>
                            <th style={{ textAlign: 'right', padding: '10px 8px' }}>Rate</th>
                            <th style={{ textAlign: 'right', padding: '10px 8px' }}>Updated</th>
                            <th style={{ textAlign: 'right', padding: '10px 8px' }}>Action</th>
                        </tr>
                    </thead>
                    <tbody>
                        {items.map(item => (
                            <tr key={item.currency_pair} style={{ borderBottom: '1px solid #f8fafc' }}>
                                <td style={{ padding: '12px 8px', fontWeight: 500 }}>{item.currency_pair}</td>
                                <td style={{ padding: '12px 8px', textAlign: 'right', fontFamily: 'monospace', fontSize: '1.1em', fontWeight: 600, color: item.rate > 0 ? '#10b981' : '#cbd5e1' }}>
                                    {item.rate > 0 ? item.rate.toFixed(4) : '-'}
                                </td>
                                <td style={{ padding: '12px 8px', textAlign: 'right', color: '#94a3b8', fontSize: '0.8em' }}>
                                    {item.updated_at > 0 ? new Date(item.updated_at * 1000).toLocaleTimeString() : '-'}
                                </td>
                                <td style={{ padding: '12px 8px', textAlign: 'right' }}>
                                    <button
                                        onClick={() => handleRemove(item.currency_pair)}
                                        style={{
                                            background: 'transparent',
                                            color: '#ef4444',
                                            border: '1px solid #ef4444',
                                            borderRadius: 4,
                                            padding: '4px 8px',
                                            cursor: 'pointer',
                                            fontSize: '0.8em',
                                            transition: 'all 0.2s'
                                        }}
                                        onMouseOver={e => { e.target.style.background = '#ef4444'; e.target.style.color = 'white' }}
                                        onMouseOut={e => { e.target.style.background = 'transparent'; e.target.style.color = '#ef4444' }}
                                    >
                                        Remove
                                    </button>
                                </td>
                            </tr>
                        ))}
                    </tbody>
                </table>
            )}
        </div>
    );
};

export default Watchlist;
