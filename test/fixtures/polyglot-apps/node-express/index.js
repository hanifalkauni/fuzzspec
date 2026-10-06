const express = require('express');
const app = express();
app.use(express.json());

// Swagger path
app.get('/swagger.json', (req, res) => {
  res.sendFile(__dirname + '/swagger.json');
});

app.post('/products', (req, res) => {
  const { title, price } = req.body;
  // Bug: Unhandled null pointer / TypeError when title is null or undefined
  if (title.length === 0) {
    return res.status(400).json({ error: 'Title required' });
  }
  res.status(201).json({ id: 1, title, price });
});

app.listen(3000);
