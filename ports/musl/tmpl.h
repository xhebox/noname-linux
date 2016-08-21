#if !defined __x86_64__
# include <__target32/bits/__name>
#endif
#if defined __x86_64__ && defined __LP64__
# include <__target64/bits/__name>
#endif
