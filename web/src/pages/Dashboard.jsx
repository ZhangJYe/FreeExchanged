
import Watchlist from '../components/Watchlist';
import RateTable from '../components/RateTable';

const Dashboard = () => {
    return (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(350px, 1fr))', gap: 20 }}>
            <div style={{ minWidth: 320 }}>
                <Watchlist />
            </div>
            <div style={{ minWidth: 320 }}>
                <RateTable />
            </div>
        </div>
    );
};
export default Dashboard;
