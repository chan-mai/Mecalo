Promise.all([
  axios.get('/data/menu/menu_category/1.json?'+mosconfig.cache10),
  axios.get('/data/menu/menu_category/7.json?'+mosconfig.cache10),
  axios.get('/data/menu/menu_category/1.json?'+mosconfig.cache10),
]);
