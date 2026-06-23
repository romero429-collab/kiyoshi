import React from 'react';
import { View, StyleSheet } from 'react-native';
import ChatScreen from './screens/ChatScreen';

const App = () => {
  return (
    <View style={styles.container}>
      <ChatScreen />
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#0a0a0a',
  },
});

export default App;
