// Auto-generated from en.json. Do not edit.
declare const messages: {
  Auth: {
    email: string;
    emailPlaceholder: string;
    password: string;
    login: string;
    loggingIn: string;
    forgotPassword: string;
    invalidCredentials: string;
    passwordTooShort: string;
    passwordsDontMatch: string;
  };
  Nav: {
    dashboard: string;
    bots: string;
    backtest: string;
    settings: string;
    apiKeys: string;
    billing: string;
    signOut: string;
  };
  Dashboard: {
    title: string;
    totalBalance: string;
    activeBots: string;
    todayPnl: string;
    recentTrades: string;
    noTrades: string;
    sessions: string;
    portfolioValue: string;
    strategies: string;
    manage: string;
    viewAll: string;
    noStrategies: string;
    createStrategy: string;
  };
  Bots: {
    title: string;
    create: string;
    start: string;
    stop: string;
    delete: string;
    backToBots: string;
    tradingDashboard: string;
    balances: string;
    openOrders: string;
    closedOrders: string;
    recentTrades: string;
    strategies: string;
    activeCount: string;
    maker: string;
    locked: string;
    errorBanner: string;
    stoppedBanner: string;
    noBalances: string;
    startToSeeData: string;
    noOpenOrders: string;
    noClosedOrders: string;
    noTrades: string;
    status: {
      running: string;
      stopped: string;
      error: string;
    };
    strategyStatus: {
      running: string;
      idle: string;
    };
    loading: string;
    noStrategies: string;
    userContainer: string;
    dashboard: string;
    remove: string;
    removeConfirm: string;
    strategiesCount: string;
    containerName: string;
    name: string;
    mode: {
      live: string;
      paper: string;
      backtest: string;
    };
    sessionRoles: string;
    cancel: string;
    creating: string;
    futures: string;
    strategyParams: string;
    errorLoading: string;
    save: string;
    saving: string;
    testnet: string;
    containerLogs: string;
  };
  Backtest: {
    title: string;
    run: string;
    results: string;
    totalProfit: string;
    maxDrawdown: string;
    sharpeRatio: string;
    winRate: string;
    trades: string;
    noResults: string;
    strategy: string;
    exchange: string;
    startDate: string;
    endDate: string;
    running: string;
    backtestOutput: string;
    backtestDuration: string;
  };
  Settings: {
    apiKeys: {
      title: string;
      add: string;
      exchange: string;
      apiKey: string;
      apiSecret: string;
      testnet: string;
      passphrase: string;
      empty: string;
      delete: string;
      verified: string;
      verificationFailed: string;
    };
  };
  Errors: {
    serverError: string;
    fetchFailed: string;
  };
};
export default messages;
