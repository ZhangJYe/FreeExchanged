
import { useEffect, useState } from 'react';
import api from '../api/client';

const RateTable = () => {
    const [rates, setRates] = useState([]);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        const fetch = async () => {
            try {
                const popularPairs = ['USD/CNY', 'EUR/USD', 'GBP/USD', 'JPY/USD', 'BTC/USD'];
                const promises = popularPairs.map(async (pair) => {
                    const [from, to] = pair.split('/');
                    try {
                        const res = await api.get('/rate', { params: { from, to } });
                        return { pair, rate: res.rate, updated_at: res.updated_at };
                    } catch (e) {
                        return { pair, rate: 0, updated_at: 0 };
                    }
                });

                const data = await Promise.all(promises);
                setRates(data);
                setLoading(false);
            } catch (e) {
                console.error(e);
            }
        };

        fetch();
        const timer = setInterval(fetch, 5000);
        return () => clearInterval(timer);
    }, []);

    return (
        <div className="card">
            <h3 style={{ marginTop: 0 }}>Popular Rates</h3>
            {loading ? <p>Loading...</p> : (
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(140px, 1fr))', gap: 15 }}>
                    {rates.map(r => (
                        <div key={r.pair} style={{
                            padding: 15,
                            background: '#f8fafc',
                            borderRadius: 8,
                            textAlign: 'center',
                            border: '1px solid #e2e8f0',
                            transition: 'transform 0.2s',
                            cursor: 'pointer'
                        }}
                            onMouseOver={e => e.currentTarget.style.transform = 'translateY(-2px)'}
                            onMouseOut={e => e.currentTarget.style.transform = 'translateY(0)'}
                        >
                            <div style={{ fontSize: '0.9em', color: '#64748b', marginBottom: 5, fontWeight: 600 }}>{r.pair}</div>
                            <div style={{ fontSize: '1.4em', fontWeight: 'bold', color: r.rate > 0 ? '#0f172a' : '#cbd5e1', fontFamily: 'monospace' }}>
                                {r.rate > 0 ? r.rate.toFixed(4) : '-'}
                            </div>
                            <div style={{ fontSize: '0.75em', color: '#94a3b8', marginTop: 5 }}>
                                {r.updated_at > 0 ? new Date(r.updated_at * 1000).toLocaleTimeString() : 'N/A'}
                            </div>
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
};

export default RateTable;
