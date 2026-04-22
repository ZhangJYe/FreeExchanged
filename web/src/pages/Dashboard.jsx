import { useEffect, useState } from 'react';
import Watchlist from '../components/Watchlist';
import RateTable from '../components/RateTable';

const Dashboard = () => {
    const [now, setNow] = useState(new Date());

    useEffect(() => {
        const timer = setInterval(() => setNow(new Date()), 1000);
        return () => clearInterval(timer);
    }, []);

    return (
        <main className="container dashboard">
            <section className="desk-header">
                <div>
                    <p className="desk-kicker">FX / WATCHLIST / HOT QUOTES</p>
                    <h2 className="desk-title">Currency desk</h2>
                </div>
                <div className="desk-clock">
                    <span className="clock-label">LOCAL SESSION</span>
                    <span className="clock-value">{now.toLocaleTimeString()}</span>
                </div>
            </section>

            <section className="market-grid">
                <Watchlist />
                <RateTable />
            </section>
        </main>
    );
};

export default Dashboard;
