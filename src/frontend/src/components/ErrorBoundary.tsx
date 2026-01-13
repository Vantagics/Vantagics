import React from 'react';

interface ErrorBoundaryState {
    hasError: boolean;
    error?: Error;
}

interface ErrorBoundaryProps {
    children: React.ReactNode;
}

class ErrorBoundary extends React.Component<ErrorBoundaryProps, ErrorBoundaryState> {
    constructor(props: ErrorBoundaryProps) {
        super(props);
        this.state = { hasError: false };
    }

    static getDerivedStateFromError(error: Error): ErrorBoundaryState {
        return { hasError: true, error };
    }

    componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
        console.error('ErrorBoundary caught an error:', error, errorInfo);
    }

    render() {
        if (this.state.hasError) {
            return (
                <div className="flex h-screen w-screen bg-slate-50 items-center justify-center flex-col gap-6">
                    <div className="text-center max-w-md px-6">
                        <h2 className="text-xl font-semibold text-red-600 mb-2">应用程序错误</h2>
                        <p className="text-sm text-slate-600 mb-4">
                            应用程序遇到了一个错误。请刷新页面重试。
                        </p>
                        {this.state.error && (
                            <details className="text-xs text-slate-500 bg-slate-100 p-3 rounded border">
                                <summary className="cursor-pointer font-medium">错误详情</summary>
                                <pre className="mt-2 text-left overflow-auto">
                                    {this.state.error.message}
                                    {this.state.error.stack && '\n\n' + this.state.error.stack}
                                </pre>
                            </details>
                        )}
                        <button
                            onClick={() => window.location.reload()}
                            className="mt-4 px-6 py-2 bg-blue-600 text-white text-sm font-medium rounded-md hover:bg-blue-700 transition-colors"
                        >
                            刷新页面
                        </button>
                    </div>
                </div>
            );
        }

        return this.props.children;
    }
}

export default ErrorBoundary;