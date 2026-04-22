import { useEffect, useState } from 'react';
import api from '../api/client';

const popularPairs = ['USD/CNY', 'EUR/USD', 'GBP/USD', 'JPY/USD', 'AUD/CAD', 'HKD/CNY'];

const SparkIcon = () => (
    <svg className="icon" viewBox="0 0 24 24" fill="none" aria-hidden="true">
        <path d="m4 15 4.5-4.5 3 3L20 5" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
        <path d="M15 5h5v5" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
);

const formatRate = (rate) => {
    if (!rate || rate <= 0) return '-';
    if (rate >= 100) return rate.toFixed(2);
    if (rate >= 10) return rate.toFixed(3);
    return rate.toFixed(4);
};

const RateTable = () => {
    const [rates, setRates] = useState([]);
    const [loading, setLoading] = useState(true);
    const [lastRefresh, setLastRefresh] = useState(null);

    useEffect(() => {
        let mounted = true;

        const fetch = async () => {
            const data = await Promise.all(popularPairs.map(async (pair) => {
                const [from, to] = pair.split('/');
                try {
                    const res = await api.get('/rate', { params: { from, to } });
                    return { pair, rate: res.rate, updated_at: res.updated_at };
                } catch {
                    return { pair, rate: 0, updated_at: 0 };
                }
            }));

            if (mounted) {
                setRates(data);
                setLastRefresh(new Date());
                setLoading(false);
            }
        };

        fetch();
        const timer = setInterval(fetch, 5000);
        return () => {
            mounted = false;
            clearInterval(timer);
        };
    }, []);

    return (
        <section className="market-panel">
            <div className="panel-head">
                <div>
                    <h3 className="panel-title">Popular rates</h3>
                    <span className="panel-meta">
                        {lastRefresh ? `REFRESH ${lastRefresh.toLocaleTimeString()}` : 'SYNCING'}
                    </span>
                </div>
                <SparkIcon />
            </div>

            <div className="panel-body">
                {loading ? (
                    <div className="empty-state">Loading quotes</div>
                ) : (
                    <div className="rate-board">
                        {rates.map((r) => (
                            <article className="rate-cell" key={r.pair}>
                                <div className="rate-topline">
                                    <span className="pair-badge">{r.pair}</span>
                                    <span className={`status-dot ${r.rate > 0 ? '' : 'offline'}`} />
                                </div>
                                <p className={`quote-number ${r.rate > 0 ? '' : 'rate-muted'}`}>
                                    {formatRate(r.rate)}
                                </p>
                                <div className="quote-time">
                                    {r.updated_at > 0 ? new Date(r.updated_at * 1000).toLocaleTimeString() : 'NO PRINT'}
                                </div>
                            </article>
                        ))}
                    </div>
                )}
            </div>
        </section>
    );
};

export default RateTable;
