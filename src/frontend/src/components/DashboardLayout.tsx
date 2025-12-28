import React from 'react';

interface DashboardLayoutProps {
    children: React.ReactNode;
}

const DashboardLayout: React.FC<DashboardLayoutProps> = ({ children }) => {
    return (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6 p-6 h-full overflow-y-auto bg-slate-50">
            {children}
        </div>
    );
};

export default DashboardLayout;
