import auth from './auth';
import foods from './foods';
import historyPage from './history';
import home from './home';
import settings from './settings';

export default {
	...home,
	...historyPage,
	...foods,
	...settings,
	...auth,
};
