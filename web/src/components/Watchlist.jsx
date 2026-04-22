import { useEffect, useState } from 'react';
import api from '../api/client';

const AddIcon = () => (
    <svg className="icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
        <path d="M12 5v14M5 12h14" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" />
    </svg>
);

const TrashIcon = () => (
    <svg className="icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
        <path d="M5 7h14M10 11v6m4-6v6M8 7l.8 12h6.4L16 7M9.5 7l.7-2h3.6l.7 2" stroke="currentColor" strokeWidth="1.7" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
);

const normalizePair = (value) => value.trim().toUpperCase().replace(/\s+/g, '');

const formatRate = (rate) => {
    if (!rate || rate <= 0) return '-';
    if (rate >= 100) return rate.toFixed(2);
    return rate.toFixed(4);
};

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
        const timer = setInterval(fetchList, 5000);
        return () => clearInterval(timer);
    }, []);

    const handleAdd = async (e) => {
        e.preventDefault();
        const pair = normalizePair(newPair);
        if (!pair) return;

        setLoading(true);
        try {
            await api.post('/watchlist/add', { currency_pair: pair });
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
        <section className="market-panel">
            <div className="panel-head">
                <div>
                    <h3 className="panel-title">My watchlist</h3>
                    <span className="panel-meta">{items.length} ACTIVE PAIRS</span>
                </div>
            </div>

            <div className="panel-body">
                {message && (
                    <div className={`notice ${message.startsWith('Error') ? 'notice-error' : 'notice-ok'}`}>
                        {message}
                    </div>
                )}

                <form className="watch-form" onSubmit={handleAdd}>
                    <input
                        className="field"
                        value={newPair}
                        onChange={(e) => setNewPair(e.target.value.toUpperCase())}
                        placeholder="USD/CNY"
                        disabled={loading}
                    />
                    <button type="submit" className="btn btn-primary" disabled={loading}>
                        <AddIcon />
                        {loading ? 'Adding' : 'Add'}
                    </button>
                </form>

                {items.length === 0 ? (
                    <div className="empty-state">No watchlist pairs</div>
                ) : (
                    <div className="data-table-wrap">
                        <table className="data-table">
                            <thead>
                                <tr>
                                    <th>PAIR</th>
                                    <th>RATE</th>
                                    <th>UPDATED</th>
                                    <th>ACTION</th>
                                </tr>
                            </thead>
                            <tbody>
                                {items.map((item) => (
                                    <tr key={item.currency_pair}>
                                        <td className="pair-code">{item.currency_pair}</td>
                                        <td className={item.rate > 0 ? 'rate-value' : 'rate-muted'}>{formatRate(item.rate)}</td>
                                        <td className="rate-muted">
                                            {item.updated_at > 0 ? new Date(item.updated_at * 1000).toLocaleTimeString() : '-'}
                                        </td>
                                        <td>
                                            <button
                                                type="button"
                                                className="btn icon-btn"
                                                onClick={() => handleRemove(item.currency_pair)}
                                                title="Remove"
                                                aria-label={`Remove ${item.currency_pair}`}
                                            >
                                                <TrashIcon />
                                            </button>
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                )}
            </div>
        </section>
    );
};

export default Watchlist;
